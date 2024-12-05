import Cypress from 'cypress';

const PORT = 3000;
const BASE_URL = `http://localhost:${PORT}`;

const config: Cypress.ConfigOptions = {
  e2e: {
    setupNodeEvents(on, config) {},
    baseUrl: BASE_URL,
    supportFile: false,
    waitForAnimations: true,
  },
};

export default Cypress.defineConfig(config);
