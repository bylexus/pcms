/**
 * Config helper module. Abstracts some paths, based on the site config file
 *
 * (c) 2019 Alexander Schenkel
 * This file is part of pcms
 */
const path = require('path');

function config(siteConfigPath) {
    const siteConfig = require(siteConfigPath);
    const sitePath = path.join(path.dirname(siteConfigPath), 'site');
    const themesPath = path.join(path.dirname(siteConfigPath), 'themes');
    const theme = siteConfig.theme || 'default';
    const themePath = path.join(themesPath, theme);

    return {
        siteConfig,
        sitePath,
        themesPath,
        theme,
        themePath
    };
}

module.exports = config;
