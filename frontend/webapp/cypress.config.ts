import Cypress from 'cypress';

const PORT = 3000;
const BASE_URL = `http://localhost:${PORT}`;

const config: Cypress.ConfigOptions = {
  e2e: {
    baseUrl: BASE_URL,
    supportFile: false,
    waitForAnimations: false,
    viewportWidth: 1920,
    viewportHeight: 1080,
    retries: {
      runMode: 1,
      openMode: 0,
    },
    setupNodeEvents(on, config) {
      on('task', {
        log: (message) => {
          console.log(message);
          return null;
        },
      });
    },
  },
};

export default Cypress.defineConfig(config);
