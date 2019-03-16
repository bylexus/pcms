#!/usr/bin/env node

const path = require('path');
const fs = require('fs-extra');
const templateDir = path.resolve(path.join(__dirname, 'site-template'));
const destDir = path.resolve(process.cwd());

console.info('--------------- pcms site generator -------------------');
console.info(` copying site template:
    from ${templateDir}
    to ${destDir}`);
console.info('-------------------------------------------------------');

fs.copy(templateDir, destDir, { overwrite: false, errorOnExist: false }, err => {
    if (err) {
        return console.error('Copy failed: ' + err);
    } else {
        console.info('Copied skeleton files. No files were overwritten.');
        console.info('Your new site is reay. Start with:\n');
        console.info('$ DEBUG=server,pcms node server.js');
    }
});
