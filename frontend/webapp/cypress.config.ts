import Cypress from 'cypress';

const config: Cypress.ConfigOptions = {
  e2e: {
    baseUrl: 'http://localhost:3000',
    setupNodeEvents(on, config) {},
    supportFile: false,
    waitForAnimations: true,
  },
};

export default Cypress.defineConfig(config);
