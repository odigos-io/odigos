import Cypress from 'cypress';

const config: Cypress.ConfigOptions = {
  e2e: {
    baseUrl: 'https://example.cypress.io',
    setupNodeEvents(on, config) {},
    supportFile: false,
    waitForAnimations: true,
  },
};

export default Cypress.defineConfig(config);
