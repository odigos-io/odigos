import { defineConfig } from 'cypress';

export default defineConfig({
  projectId: process.env.CYPRESS_PROJECT_ID,
  env: {
    CYPRESS_RECORD_KEY: process.env.CYPRESS_RECORD_KEY,
  },
  e2e: {
    setupNodeEvents(on, config) {
      // implement node event listeners here
    },
  },
});
