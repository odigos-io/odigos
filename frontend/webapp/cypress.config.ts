import Cypress from 'cypress';

const config: Cypress.ConfigOptions = {
  e2e: {
    // this uses the "production" build, if you want to use the "development" build, you can use "port=3000" instead
    baseUrl: 'http://localhost:3001',
    setupNodeEvents(on, config) {},
    supportFile: false,
    waitForAnimations: true,
  },
};

export default Cypress.defineConfig(config);
