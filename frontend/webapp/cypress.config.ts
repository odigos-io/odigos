import Cypress from 'cypress';
import fs from 'fs';

const PORT = 3000;
const BASE_URL = `http://localhost:${PORT}`;

const config: Cypress.ConfigOptions = {
  trashAssetsBeforeRuns: true,
  screenshotOnRunFailure: true,
  video: true,

  e2e: {
    baseUrl: BASE_URL,
    supportFile: false,
    waitForAnimations: true,
    animationDistanceThreshold: 5,
    viewportWidth: 1920,
    viewportHeight: 1080,
    defaultCommandTimeout: 10000,
    pageLoadTimeout: 30000,
    requestTimeout: 10000,
    responseTimeout: 10000,
    retries: { runMode: 0, openMode: 0 },
    allowCypressEnv: false,
    // Keep test isolation enabled but handle navigation carefully
    // Next.js 15 App Router uses client-side navigation which requires proper waits
    testIsolation: true,
    setupNodeEvents(on, config) {
      on('task', {
        log: (message) => {
          console.log(message);
          return null;
        },
      });

      on('after:spec', (spec: Cypress.Spec, results: CypressCommandLine.RunResult) => {
        if (results && results.video) {
          // Do we have failures for any retry attempts?
          const failures = results.tests.some((test) => test.attempts.some((attempt) => attempt.state === 'failed'));

          if (!failures) {
            // delete the video if the spec passed and no tests retried
            fs.unlinkSync(results.video);
          }
        }
      });
    },
  },
};

export default Cypress.defineConfig(config);
