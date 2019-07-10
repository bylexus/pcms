/**
 * internally used tools
 *
 * (c) 2019 Alexander Schenkel
 * This file is part of pcms
 */
const path = require('path');
const fs = require('fs');
const marked = require('marked');
const debug = require('debug')('pcms');

function regexpEscape(string) {
    return string.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&');
}

/**
 * Determines a page's index (or main content) file:
 * - if indexFile is given, we check if the file (pagePath/indexFile)
 *   exists and return indexFile
 * - if indexFile is not given, we check if we can find one of index.[html|md|json],
 *   and return that name.
 * - if no index file at all can be found, we return an Error.
 *
 * Returns a Promise that resolves with the index file name, or rejects with an error
 */
async function getPageIndex(pagePath, indexFile) {
    let indexFiles = ['index.html', 'index.md', 'index.json'];
    if (indexFile) {
        indexFiles = [indexFile];
    }

    for (let file of indexFiles) {
        try {
            let indexFile = path.join(pagePath, file);
            let exists = await new Promise((resolve, reject) => {
                fs.access(indexFile, fs.constants.R_OK, err => {
                    if (err) {
                        reject();
                    } else {
                        resolve(true);
                    }
                });
            });
            if (exists === true) {
                return file;
            }
        } catch (err) {
            // noop
        }
    }
    throw new Error('No index file found.');
}

/**
 * Reads the markdown content of the given file and renders it to HTML.
 *
 * Returns a promise that resolved the HTML as string
 */
function renderMarkdownFile(file) {
    return new Promise((resolve, reject) => {
        fs.readFile(file, { encoding: 'utf8' }, (err, data) => {
            if (err) {
                reject(err);
            } else {
                resolve(data);
            }
        });
    }).then(data => {
        return marked(data, {});
    });
}

/**
 * Creates the page tree, including meta information for all available
 * pages in the "sites" folder.
 *
 * returns a promise that returns the root node
 */
function parsePage(serverConfig, fullPagePath, depth = 0) {
    let siteRoot = serverConfig.sitePath;

    return new Promise((resolve, reject) => {
        const pageConfigPath = path.join(fullPagePath, 'page.json');
        fs.access(pageConfigPath, fs.constants.R_OK, async err => {
            if (err) {
                resolve(null);
            } else {
                const pageInfo = {
                    depth: depth,
                    fullPath: fullPagePath,
                    routePart: path.basename(fullPagePath) || '/',
                    route: fullPagePath.replace(new RegExp('^' + siteRoot), '') || '/',
                    pageConfig: require(pageConfigPath),
                    pageIndex: null,
                    childPages: [],
                    childPagesByRoute: {},
                    template: null,
                    type: 'html'
                };

                // fix the page config:
                // set the 'enabled' property correctly:
                pageInfo.pageConfig.enabled = 'enabled' in pageInfo.pageConfig ? pageInfo.pageConfig.enabled : true;
                // initialize the preventDelivery property:
                pageInfo.pageConfig.preventDelivery = pageInfo.pageConfig.preventDelivery || [];
                // load the preprocessor script, and add the preprocessor script to the preventDelivery array, as regex:
                if (pageInfo.pageConfig.preprocessor) {
                    pageInfo.pageConfig.preventDelivery.push(
                        '^' + regexpEscape(pageInfo.pageConfig.preprocessor) + '$'
                    );
                    // load the preprocessor script: Note: It MUST export a function:
                    pageInfo.pageConfig.preprocessor = require(path.join(
                        fullPagePath,
                        pageInfo.pageConfig.preprocessor
                    ));
                }

                try {
                    // find the index file:
                    const pageIndex = await getPageIndex(fullPagePath, pageInfo.pageConfig.index || null);
                    const extension = pageIndex
                        .split('.')
                        .splice(1)
                        .join('.');
                    pageInfo.pageIndex = pageIndex;
                    pageInfo.template = `.${pageInfo.route}/${pageIndex}`;

                    if (extension === 'md') {
                        pageInfo.template =
                            `.${pageInfo.route}/${pageInfo.pageConfig.template}` || 'markdown-template.md';
                        pageInfo.type = 'markdown';
                    } else if (extension === 'json') {
                        pageInfo.template = `.${pageInfo.route}/${pageInfo.pageConfig.template}`;
                        pageInfo.type = 'json';
                    } else if (extension === 'js') {
                        pageInfo.type = 'js';
                        pageInfo.template = null;
                    }
                    let dirs = await readdir(fullPagePath);
                    for (let dir of dirs) {
                        let child = await parsePage(serverConfig, dir, depth + 1);
                        if (child) {
                            pageInfo.childPages.push(child);
                            pageInfo.childPagesByRoute[child.route] = child;
                        }
                    }
                    pageInfo.childPages.sort((child1, child2) => {
                        let s1 = child1.pageConfig.order;
                        let s2 = child2.pageConfig.order;
                        if (s1 < s2) {
                            return -1;
                        }
                        if (s1 > s2) {
                            return 1;
                        }
                        return 0;
                    });
                    resolve(pageInfo);
                } catch (err) {
                    return resolve(null);
                }
            }
        });
    });
}

/**
 * Reads dir entries from a given dir
 *
 * @return Promise The promise resolves with a list of dirs
 */
function readdir(dir) {
    return new Promise((resolve, reject) => {
        fs.readdir(dir, { withFileTypes: true }, (err, files) => {
            if (err) {
                reject(err);
            } else {
                resolve(files.filter(f => f.isDirectory()).map(f => path.join(dir, f.name)));
            }
        });
    });
}

module.exports = {
    getPageIndex,
    renderMarkdownFile,
    parsePage,
    debug,
    regexpEscape
};
