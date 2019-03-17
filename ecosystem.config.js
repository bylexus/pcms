module.exports = {
    apps: [
        {
            name: 'pcms-doc',
            script: 'server.js',

            // Options reference: https://pm2.io/doc/en/runtime/reference/ecosystem-file/
            instances: 1,
            autorestart: true,
            watch: process.env.NODE_ENV !== 'production' ? ['server.js', 'site'] : false,
            max_memory_restart: '1G',
            env: {
                NODE_ENV: 'development'
            },
            env_production: {
                NODE_ENV: 'production'
            }
        }
    ]
};
