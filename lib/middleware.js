/**
 * Middleware functions used in the express server part
 *
 * (c) 2019 Alexander Schenkel
 * This file is part of pcms
 */
const path = require('path');
const nunjucks = require('nunjucks');
const basicAuth = require('basic-auth');
const bcrypt = require('bcrypt');
const tools = require('./tools');
const debug = tools.debug;

/**
 * Finds a page entry in the page tree, by a given route.
 * The tree is walked down as long as the route matches.
 *
 * The matching node is returned, as well as a possible URL tail, or null.
 */
function findPageInfo(rootNode, routeParts) {
    let treeNode = rootNode;
    let foundNode = null;
    let part = routeParts.shift();
    if (part) {
        if (treeNode.routePart === part) {
            let urlTail = routeParts.join('/');

            // if the remaining part of the URL is the same as the found's node index page,
            // we stop here: The index page will not be delivered to the browser,
            // it is processed by a template engine instead.
            if (urlTail === treeNode.pageIndex) {
                routeParts.shift();
                urlTail = null;
            }

            foundNode = Object.assign({ urlTail: urlTail || null }, treeNode);

            if (treeNode.childPages.length > 0 && routeParts.length > 0) {
                for (let child of treeNode.childPages) {
                    let foundChildNode = findPageInfo(child, [].concat(routeParts));
                    if (foundChildNode) {
                        return foundChildNode;
                    }
                }
                return foundNode;
            } else {
                return foundNode;
            }
        }
    }
    return null;
}

/**
 * Checks page authentication. This middleware requires the
 * injectPageNodeMiddleware to be done already. It needs
 * the pageConfig's requiredUsers property.
 */
function authenticationMiddleware() {
    const deniedFn = function denied(res) {
        res.statusCode = 401;
        res.set('WWW-Authenticate', 'Basic realm="example"');
        res.end('Access denied');
    };
    return async (req, res, next) => {
        const config = req.app.serverConfig;
        const siteConfig = config.siteConfig;

        if (req.pageNode && req.pageNode.pageConfig.requiredUsers) {
            const authInfo = basicAuth(req);
            if (authInfo) {
                const validUsers = req.pageNode.pageConfig.requiredUsers || [];
                const users = siteConfig.users || {};
                const passwordHash = users[authInfo.name] || '';

                if (validUsers.length < 1) {
                    return next('No users defined for this page.');
                }
                debug('Checking credentials for ' + req.route);

                // Check for valid credentials: either a matching username/password,
                // or any valid user:
                let result = await bcrypt.compare(authInfo.pass, passwordHash);
                if (result && (validUsers.includes(authInfo.name) || validUsers.includes('valid-user'))) {
                    debug('Valid credentials for ' + req.route);
                    return next();
                }
            }
            debug('Access denied: invalid credentials for ' + req.route);
            return deniedFn(res);
        } else {
            next();
        }
    };
}

/**
 * Figures out the requested page / route and attaches the info to the request object,
 * assigning 'pageNode' and 'route' properties to the express request object.
 */
function injectPageNodeMiddleware(expressApp) {
    const config = expressApp.serverConfig;
    const webrootRegexp = new RegExp('^' + config.siteConfig.webroot);
    let notAvailableError = new Error('Page not available');
    notAvailableError.statusCode = 403;

    return async (req, res, next) => {
        try {
            // Find matching page node in page tree:
            const route =
                req.path
                    .replace(/\/{2,}/g, '/')
                    .replace(webrootRegexp, '')
                    .replace(/\/*$/, '') || '/';

            const routeParts = ['/'].concat(route.split('/').filter(p => p.length > 0));
            let pageNode = findPageInfo(expressApp.siteRoot, routeParts);

            // If the page is not enabled, return a 'not available' error:
            if (pageNode && 'enabled' in pageNode.pageConfig && pageNode.pageConfig.enabled !== true) {
                return next(notAvailableError);
            }

            // Do not deliver sub-pages (urlTail) that match preventDelivery entries
            if (pageNode && pageNode.pageConfig.preventDelivery.length > 0) {
                for (let entry of pageNode.pageConfig.preventDelivery) {
                    let re = new RegExp(entry);
                    if ((pageNode.urlTail || '').match(re)) {
                        return next(notAvailableError);
                    }
                }
            }
            req.pageNode = pageNode;
            req.route = route;
            next();
        } catch (err) {
            next(err);
        }

        // Check for authentication information:
        // if (pageNode && pageNode.pageConfig.requiredUsers) {
        // }
    };
}

/**
 * This is the main workhorse of the framework: It renders the requested route, if the route matches a page.
 *
 * The function itself is a factory function which need to be called with the expressjs app.
 * It then returns an expressjs middleware function.
 */
function renderPageMiddleware(expressApp) {
    const config = expressApp.serverConfig;

    nunjucks.configure([config.sitePath, path.join(config.themePath, 'templates')], {
        watch: false,
        noCache: process.env !== 'production',
        express: expressApp
    });

    return async (req, res, next) => {
        try {
            const pageNode = req.pageNode;

            // If we have a preprocessor, execute it (expect a promise):
            let preprocessorData = {};
            if (pageNode && pageNode.pageConfig.preprocessor instanceof Function) {
                preprocessorData = await pageNode.pageConfig.preprocessor(pageNode, expressApp.siteRoot);
            }
            preprocessorData = preprocessorData || {};

            // If no matching page could be found, or a deeper route is requested, hand it over to the next
            // middlewares:
            if (!pageNode || pageNode.urlTail) {
                return next();
            }

            // we HAVE found a page node, so deliver it:
            debug('Page found: ', pageNode.route, pageNode.fullPath);
            const pagePath = pageNode.fullPath;

            // the context is assigned to the template engine as additional data:
            const context = Object.assign(
                {
                    site: config.siteConfig,
                    base: config.siteConfig.webroot,
                    route: pageNode.route, // route of the node, so without the index page appended
                    fullRoute: req.route, // full URL route, including e.g. index.html
                    page: pageNode,
                    rootPage: expressApp.siteRoot,
                    now: new Date()
                },
                preprocessorData
            );
            let template = pageNode.template;

            // deliver the page according to its type:
            if (pageNode.type === 'markdown') {
                context.content = await tools.renderMarkdownFile(path.join(pagePath, pageNode.pageIndex));
                context.content = nunjucks.renderString(context.content, context);
            } else if (pageNode.type === 'json') {
                context.content = require(path.join(pagePath, pageNode.pageIndex));
            } else if (pageNode.type === 'js') {
                let requestHandler = require(path.join(pagePath, pageNode.pageIndex));
                if (!(requestHandler instanceof Function)) {
                    return next(new Error('Page does not define a request hander.'));
                }
                req.context = context;
                return requestHandler(req, res, next);
            }
            res.render(template, context);
        } catch (err) {
            next(err);
        }
    };
}

/**
 * The default error hander: It renders errors that are passed to the pipeline with next(error).
 *
 * The error handler looks for an error.html template in the theme directory.
 */
function defaultErrorHandler(err, req, res, next) {
    debug(err);
    // Map a 404 error to a standard response:
    if (err && err.statusCode === 404) {
        err = new Error('Page not found');
        err.statusCode = 404;
    }
    const config = req.app.serverConfig;
    const context = Object.assign(
        {
            site: config.siteConfig,
            base: config.siteConfig.webroot,
            route: req.path, // route of the node, so without the index page appended
            fullRoute: req.path, // full URL route, including e.g. index.html
            page: null,
            rootPage: req.app.siteRoot,
            now: new Date(),
            error: err,
            statusCode: err.statusCode || 500
        },
        {}
    );
    res.status(context.statusCode).render('error.html', context);
}

module.exports = {
    injectPageNodeMiddleware,
    authenticationMiddleware,
    renderPageMiddleware,
    defaultErrorHandler
};
