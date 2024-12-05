import Cypress from 'cypress';

// PORT=3001 uses the "production" build (served with Go)
// PORT=3000 uses the "development" build (served with Next.js)
// We have to use the "production" build when pushing to GitHub, feel free to change this for local tests...
const PORT = 3001;
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
