import Cypress from 'cypress';
import { exec as execCb } from 'child_process';
import { promisify } from 'util';
import fs from 'fs';

const execAsync = promisify(execCb);

const PORT = 3000;
const BASE_URL = `http://localhost:${PORT}`;

type ExecTaskArgs = {
  command: string;
  failOnNonZeroExit?: boolean;
};

type ExecTaskResult = {
  code: number;
  stdout: string;
  stderr: string;
};

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
        // Cypress 16 removed cy.exec(); shell out via task instead.
        // https://on.cypress.io/task
        // Match legacy cy.exec() by trimming trailing newlines from stdout/stderr.
        async exec({ command, failOnNonZeroExit = true }: ExecTaskArgs): Promise<ExecTaskResult> {
          const normalize = (s: string | undefined | null) => (s ?? '').replace(/\n+$/, '');
          try {
            const { stdout, stderr } = await execAsync(command, {
              shell: '/bin/bash',
              maxBuffer: 10 * 1024 * 1024,
            });
            return { code: 0, stdout: normalize(stdout), stderr: normalize(stderr) };
          } catch (error: unknown) {
            const err = error as { code?: number; stdout?: string; stderr?: string; message?: string };
            const result: ExecTaskResult = {
              code: typeof err.code === 'number' ? err.code : 1,
              stdout: normalize(err.stdout?.toString()),
              stderr: normalize(err.stderr?.toString() ?? err.message),
            };
            if (failOnNonZeroExit) {
              throw new Error(`exec failed (${result.code}): ${command}\nstderr: ${result.stderr}\nstdout: ${result.stdout}`);
            }
            return result;
          }
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
