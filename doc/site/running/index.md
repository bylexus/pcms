# Running the site

pcms itself does not come with any pre-configured run procedure: You just can start server.js using node.

That said, there are several options on how you can start the server in a more sophisticated way, but those are
not part of pcms, but of the available standard tools out there.

## Using `nodemon` to watch and restart

You can use `nodemon` to watch your site content and restart the server as soon as it has changed. For that,
you need to install `nodemon` first:

```sh
$ npm install --save nodemon
```

Then you can start the server and watch for changes:

```sh
$ DEBUG=server,pcms npx nodemon -w server.js -w site/ -w themes/pcms-doc/ server.js
```

This command starts the server, and restarts it as soon as changes are made either in the site folder or here, as an example,
in the `themes/pcms-doc/` folder.

## Using `pm2` for a production solution

If you want a more sophisticated tool for watching / restarting your site, you can use [pm2](http://pm2.keymetrics.io/)
as a process manager. Setting up pm2 needs some more config, though.

Install it with:

```sh
$ npm install --save pm2
```

### create an `ecosystem.config.js`

The main config of pm2 is done in a JS file called `ecosystem.config.js`. An example:

```javascript
module.exports = {
    apps: [
        {
            name: 'my-site',
            script: 'server.js',
            // Options reference: https://pm2.io/doc/en/runtime/reference/ecosystem-file/
            instances: 1,
            autorestart: true,
            watch: process.env.NODE_ENV !== 'production' ? ['server.js', 'themes', 'site'] : false,
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
```

Then you can start the site using the following command for example:

```sh
$ pm2 start ecosystem.config.js --env production
```

and watch your application status:

```sh
$ pm2 monit   # to monitor your app
$ pm2 stop    # shut down your app
$ pm2 restart # restart the server
```
