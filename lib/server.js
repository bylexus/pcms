/**
 * This is the main server that starts the express app.
 * It can be required in site bootstrap scripts, like so:
 *
 * <code>
 * const server = require('pcms');
 * server
 *     // give the full path to your site-config.json: The server needs it to gather information
 *     // about your directory origin:
 *     .start(__dirname + '/site-config.json'))
 *     .then(app => {
 *         debug(`Site is now serving at port: ${app.serverConfig.port}`);
 *         if (process.env.NODE_ENV !== 'production') {
 *             debug('Full site index:', app.siteRoot);
 *         }
 *     })
 *     .catch(err => {
 *         debug('ERROR: ', err);
 *     });
 * </code>
 *
 * (c) 2019 Alexander Schenkel
 * This file is part of pcms
 */
const path = require('path');
const compression = require('compression');
const express = require('express');
const app = express();
const router = express.Router();

const configHelper = require('./config');
const middleware = require('./middleware');
const tools = require('./tools');
const debug = tools.debug;

/**
 * This is the start function: It starts up the server.
 * It needs the path to the site-config.json file: This path is the base information the
 * server needs to extract all other relative paths.
 *
 * @return Promise The promise is resolved with the expressjs app once the server has started.
 */
function start(siteConfigPath) {
    const config = configHelper(siteConfigPath);
    // app-wide config is injected onto express app:
    app.serverConfig = config;

    return new Promise((resolve, reject) => {
        // gzip compression
        app.use(compression());

        /**
         * The page rendering middleware. Note that the order is important:
         * 1. injectPageNodeMiddleware parses the route and injects a pageNode object to the request
         * 2. the authentication middleware now can check if the authentication is correct / needed for that page
         * 3. the page is finally rendered using renderPageMiddleware
         */
        router.use(
            middleware.injectPageNodeMiddleware(app),
            middleware.authenticationMiddleware(),
            middleware.renderPageMiddleware(app)
        );

        // prevent the delivery of page.json config files:
        router.all('*/page.json', (req, res, next) => {
            const err = new Error('Page not found');
            err.statusCode = 404;
            throw err;
        });

        // static route for actual theme static folder
        router.use(
            config.siteConfig.webroot + '/theme/static',
            express.static(path.join(config.themePath, 'static'), { fallthrough: false, index: false })
        );

        // static route for site-folder statics
        router.use(express.static('site', { fallthrough: false, index: false }));

        // default error handler:
        router.use(middleware.defaultErrorHandler);

        // register main router:
        app.use(config.siteConfig.webroot, router);

        debug('Building page tree ...');
        tools
            .parsePage(config, config.sitePath)
            .then(rootInfo => {
                // debug(rootInfo);
                debug('Page tree complete. Starting now.');
                // Storing the page root on the app object, to make it gloablly available:
                app.siteRoot = rootInfo;
                app.siteRoot.routePart = '/'; // The root node always is called '/'
                app.listen(config.siteConfig.port, () => {
                    debug(`server listening on port ${config.siteConfig.port}!`);
                    resolve(app);
                });
            })
            .catch(err => {
                debug(err);
                reject(err);
            });
    });
}

module.exports = {
    start,
    expressApp: app
};
