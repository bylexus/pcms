const server = require('pcms');
const path = require('path');
const debug = require('debug')('server');

/**
 * Start the server, which is an expressjs app. You can get the express app instance if you
 * need to configure things in advance:
 *
 * const app = server.expressApp;
 */
server
    // give the full path to your site-config.json: The server needs it to gather information
    // about your directory origin:
    .start(path.join(__dirname, 'site-config.json'))
    .then(app => {
        debug(`Site is now serving at port: ${app.serverConfig.siteConfig.port}`);
    })
    .catch(err => {
        debug('ERROR: ', err);
    });
