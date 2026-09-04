import { buildApp } from './app.js';
import { loadConfig } from './config.js';

const config = loadConfig();

buildApp(config)
  .then(async (app) => {
    await app.listen({ port: config.PORT, host: config.HOST });
  })
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });
