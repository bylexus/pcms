#!/usr/bin/env node

const bcrypt = require('bcrypt');
let pw = process.argv.length === 3 ? process.argv[2] : null;

if (!pw) {
    console.error('Please give ONE Password as argument.');
    process.exit(1);
}

let hash = bcrypt.hashSync(pw, 10);
console.info(`Bcrypted Password: ${hash}`);
