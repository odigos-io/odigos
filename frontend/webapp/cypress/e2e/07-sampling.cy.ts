import { CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, TEXTS } from '../constants';
import { aliasQuery, awaitToast, getCrdIds, handleExceptions, hasOperationName, visitPage, waitForGraphqlOperation } from '../functions';

const namespace = NAMESPACES.ODIGOS;
const crdName = CRD_NAMES.SAMPLING;

const NOISY_RULE_NAME = 'Cypress Noisy Rule';
const HIGHLY_RELEVANT_RULE_NAME = 'Cypress Highly Relevant Rule';
const COST_REDUCTION_RULE_NAME = 'Cypress Cost Reduction Rule';
const UPDATED_SUFFIX = ' Updated';

describe('Sampling Rules CRUD', () => {
  beforeEach(() => {
    cy.intercept('/graphql', (req) => {
      if (hasOperationName(req, 'GetConfig')) {
        aliasQuery(req, 'GetConfig');

        req.reply((res) => {
          res.body.data = { config: { tier: 'onprem' } };
        });
      }
    }).as('gql');

    handleExceptions();
  });

  // ── Cleanup ──────────────────────────────────────────────────────────────

  it('Should clean up any existing sampling CRDs before tests', () => {
    cy.exec(`kubectl delete ${crdName} --all -n ${namespace}`, { failOnNonZeroExit: false });
    cy.wait(2000);
  });

  // ── CREATE: Noisy Operation Rule ─────────────────────────────────────────

  it('Should create a Noisy Operation rule with HTTP server route', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        // Already on Noisy Operations tab by default
        cy.get(DATA_IDS.SAMPLING_BTN_CREATE_RULE).click();

        // Fill rule name
        cy.get('input[name=sampling-input-rule-name]').type(NOISY_RULE_NAME);
        cy.get('input[name=sampling-input-note]').type('Created by Cypress');

        // Select HTTP server operation type
        cy.contains('button', 'HTTP server').click();

        // Fill route (radio "route" should be selected by default)
        cy.get('input[name=sampling-input-route]').type('api/health');

        // Set percentage mode to "sample" and pick 1%
        cy.contains('button', 'sample').click();
        cy.contains('button', '1%').click();

        // Submit the form
        cy.get(DATA_IDS.SAMPLING_CREATE_BTN_SUBMIT).click();

        waitForGraphqlOperation('CreateNoisyOperationRule').then(() => {
          awaitToast({ message: 'Successfully created' });
        });
      });
    });
  });

  // ── CREATE: Highly Relevant Operation Rule ───────────────────────────────

  it('Should create a Highly Relevant Operation rule (Error type)', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        // Switch to Highly Relevant Operations tab
        cy.contains('button', 'Highly Relevant Operations').click();
        cy.wait(500);

        cy.get(DATA_IDS.SAMPLING_BTN_CREATE_RULE).click();

        // Fill rule name
        cy.get('input[name=sampling-input-rule-name]').type(HIGHLY_RELEVANT_RULE_NAME);
        cy.get('input[name=sampling-input-note]').type('Created by Cypress');

        // Select rule type: Error
        cy.contains('button', 'Error').click();

        // Keep percentage mode as "keep all" (default for highly relevant)

        // Submit the form
        cy.get(DATA_IDS.SAMPLING_CREATE_BTN_SUBMIT).click();

        waitForGraphqlOperation('CreateHighlyRelevantOperationRule').then(() => {
          awaitToast({ message: 'Successfully created' });
        });
      });
    });
  });

  // ── CREATE: Cost Reduction Rule ──────────────────────────────────────────

  it('Should create a Cost Reduction rule', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        // Switch to Cost Reduction tab
        cy.contains('button', 'Cost Reduction').click();
        cy.wait(500);

        cy.get(DATA_IDS.SAMPLING_BTN_CREATE_RULE).click();

        // Fill rule name
        cy.get('input[name=sampling-input-rule-name]').type(COST_REDUCTION_RULE_NAME);
        cy.get('input[name=sampling-input-note]').type('Created by Cypress');

        // Keep default operation (all operations)
        // Cost Reduction defaults to "sample" mode with 50% preset
        cy.contains('button', '25%').click();

        // Submit the form
        cy.get(DATA_IDS.SAMPLING_CREATE_BTN_SUBMIT).click();

        waitForGraphqlOperation('CreateCostReductionRule').then(() => {
          awaitToast({ message: 'Successfully created' });
        });
      });
    });
  });

  // ── VERIFY CREATE via kubectl ────────────────────────────────────────────

  it('Should have sampling CRDs in the cluster after creation', () => {
    cy.exec(`kubectl get ${crdName} -n ${namespace} -o json`).then(({ stdout }) => {
      const parsed = JSON.parse(stdout);
      const items = parsed.items || [];
      expect(items.length).to.be.greaterThan(0);

      const sampling = items[0];
      const spec = sampling.spec;

      // Verify noisy operation rule exists
      const noisyRules = spec.noisyOperationRules || [];
      const noisyRule = noisyRules.find((r: { name: string }) => r.name === NOISY_RULE_NAME);
      expect(noisyRule, `Expected noisy rule "${NOISY_RULE_NAME}" to exist`).to.not.be.undefined;
      expect(noisyRule.httpServerRoute).to.eq('api/health');

      // Verify highly relevant operation rule exists
      const hrRules = spec.highlyRelevantOperationRules || [];
      const hrRule = hrRules.find((r: { name: string }) => r.name === HIGHLY_RELEVANT_RULE_NAME);
      expect(hrRule, `Expected highly relevant rule "${HIGHLY_RELEVANT_RULE_NAME}" to exist`).to.not.be.undefined;

      // Verify cost reduction rule exists
      const crRules = spec.costReductionRules || [];
      const crRule = crRules.find((r: { name: string }) => r.name === COST_REDUCTION_RULE_NAME);
      expect(crRule, `Expected cost reduction rule "${COST_REDUCTION_RULE_NAME}" to exist`).to.not.be.undefined;
    });
  });

  // ── READ: Verify rules appear in the UI ──────────────────────────────────

  it('Should display the Noisy Operation rule in the UI', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        // Noisy tab is default
        cy.contains(NOISY_RULE_NAME).should('exist');
      });
    });
  });

  it('Should display the Highly Relevant Operation rule in the UI', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        cy.contains('button', 'Highly Relevant Operations').click();
        cy.wait(500);
        cy.contains(HIGHLY_RELEVANT_RULE_NAME).should('exist');
      });
    });
  });

  it('Should display the Cost Reduction rule in the UI', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        cy.contains('button', 'Cost Reduction').click();
        cy.wait(500);
        cy.contains(COST_REDUCTION_RULE_NAME).should('exist');
      });
    });
  });

  // ── UPDATE: Edit the Noisy Operation rule ────────────────────────────────

  it('Should update the Noisy Operation rule name', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        // Click the rule row to open view drawer
        cy.contains(NOISY_RULE_NAME).click();

        // Click Edit button in the view drawer footer
        cy.get(DATA_IDS.SAMPLING_VIEW_BTN_EDIT).click();

        // Update the rule name
        cy.get('input[name=sampling-input-rule-name]').click().focused().clear().type(NOISY_RULE_NAME + UPDATED_SUFFIX);

        // Save the edit
        cy.get(DATA_IDS.SAMPLING_VIEW_EDIT_BTN_SAVE).click();

        waitForGraphqlOperation('UpdateNoisyOperationRule').then(() => {
          awaitToast({ message: 'Successfully updated' });
        });
      });
    });
  });

  // ── UPDATE: Edit the Highly Relevant rule ────────────────────────────────

  it('Should update the Highly Relevant Operation rule name', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        cy.contains('button', 'Highly Relevant Operations').click();
        cy.wait(500);

        // Click the rule row to open view drawer
        cy.contains(HIGHLY_RELEVANT_RULE_NAME).click();

        // Click Edit button
        cy.get(DATA_IDS.SAMPLING_VIEW_BTN_EDIT).click();

        // Update the rule name
        cy.get('input[name=sampling-input-rule-name]').click().focused().clear().type(HIGHLY_RELEVANT_RULE_NAME + UPDATED_SUFFIX);

        // Save the edit
        cy.get(DATA_IDS.SAMPLING_VIEW_EDIT_BTN_SAVE).click();

        waitForGraphqlOperation('UpdateHighlyRelevantOperationRule').then(() => {
          awaitToast({ message: 'Successfully updated' });
        });
      });
    });
  });

  // ── UPDATE: Edit the Cost Reduction rule ─────────────────────────────────

  it('Should update the Cost Reduction rule name', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        cy.contains('button', 'Cost Reduction').click();
        cy.wait(500);

        // Click the rule row to open view drawer
        cy.contains(COST_REDUCTION_RULE_NAME).click();

        // Click Edit button
        cy.get(DATA_IDS.SAMPLING_VIEW_BTN_EDIT).click();

        // Update the rule name
        cy.get('input[name=sampling-input-rule-name]').click().focused().clear().type(COST_REDUCTION_RULE_NAME + UPDATED_SUFFIX);

        // Save the edit
        cy.get(DATA_IDS.SAMPLING_VIEW_EDIT_BTN_SAVE).click();

        waitForGraphqlOperation('UpdateCostReductionRule').then(() => {
          awaitToast({ message: 'Successfully updated' });
        });
      });
    });
  });

  // ── VERIFY UPDATE via kubectl ────────────────────────────────────────────

  it('Should have updated rule names in the cluster', () => {
    cy.exec(`kubectl get ${crdName} -n ${namespace} -o json`).then(({ stdout }) => {
      const parsed = JSON.parse(stdout);
      const items = parsed.items || [];
      expect(items.length).to.be.greaterThan(0);

      const spec = items[0].spec;

      const noisyRule = (spec.noisyOperationRules || []).find((r: { name: string }) => r.name === NOISY_RULE_NAME + UPDATED_SUFFIX);
      expect(noisyRule, `Expected updated noisy rule name`).to.not.be.undefined;

      const hrRule = (spec.highlyRelevantOperationRules || []).find((r: { name: string }) => r.name === HIGHLY_RELEVANT_RULE_NAME + UPDATED_SUFFIX);
      expect(hrRule, `Expected updated highly relevant rule name`).to.not.be.undefined;

      const crRule = (spec.costReductionRules || []).find((r: { name: string }) => r.name === COST_REDUCTION_RULE_NAME + UPDATED_SUFFIX);
      expect(crRule, `Expected updated cost reduction rule name`).to.not.be.undefined;
    });
  });

  // ── VERIFY UPDATE in UI ──────────────────────────────────────────────────

  it('Should display updated Noisy rule name in the UI', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        cy.contains(NOISY_RULE_NAME + UPDATED_SUFFIX).should('exist');
      });
    });
  });

  // ── DELETE: Delete the Noisy Operation rule ──────────────────────────────

  it('Should delete the Noisy Operation rule via the view drawer', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        // Click the rule to open view drawer
        cy.contains(NOISY_RULE_NAME + UPDATED_SUFFIX).click();

        // Click delete in the view drawer footer
        cy.get(DATA_IDS.SAMPLING_VIEW_BTN_DELETE).click();

        // Confirm the delete modal
        cy.contains('button', 'Delete').click();

        waitForGraphqlOperation('DeleteNoisyOperationRule').then(() => {
          awaitToast({ message: 'Successfully deleted' });
        });
      });
    });
  });

  // ── DELETE: Delete the Highly Relevant rule via view drawer ────────────────

  it('Should delete the Highly Relevant Operation rule via the view drawer', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        cy.contains('button', 'Highly Relevant Operations').click();
        cy.wait(500);

        // Click the rule to open view drawer
        cy.contains(HIGHLY_RELEVANT_RULE_NAME + UPDATED_SUFFIX).click();

        // Click delete in the view drawer footer
        cy.get(DATA_IDS.SAMPLING_VIEW_BTN_DELETE).click();

        // Confirm the delete modal
        cy.contains('button', 'Delete').click();

        waitForGraphqlOperation('DeleteHighlyRelevantOperationRule').then(() => {
          awaitToast({ message: 'Successfully deleted' });
        });
      });
    });
  });

  // ── DELETE: Delete the Cost Reduction rule ───────────────────────────────

  it('Should delete the Cost Reduction rule via the view drawer', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        cy.contains('button', 'Cost Reduction').click();
        cy.wait(500);

        // Click the rule to open view drawer
        cy.contains(COST_REDUCTION_RULE_NAME + UPDATED_SUFFIX).click();

        // Click delete in the view drawer footer
        cy.get(DATA_IDS.SAMPLING_VIEW_BTN_DELETE).click();

        // Confirm the delete modal
        cy.contains('button', 'Delete').click();

        waitForGraphqlOperation('DeleteCostReductionRule').then(() => {
          awaitToast({ message: 'Successfully deleted' });
        });
      });
    });
  });

  // ── VERIFY DELETE via kubectl ────────────────────────────────────────────

  it('Should have no user-created sampling rules in the cluster after deletion', () => {
    cy.exec(`kubectl get ${crdName} -n ${namespace} -o json`, { failOnNonZeroExit: false }).then(({ stdout, stderr }) => {
      if (stderr.includes('No resources found') || !stdout.trim()) {
        // No sampling CRDs at all — that's fine
        return;
      }

      const parsed = JSON.parse(stdout);
      const items = parsed.items || [];

      if (items.length === 0) return;

      const spec = items[0].spec;
      const noisyRules = spec.noisyOperationRules || [];
      const hrRules = spec.highlyRelevantOperationRules || [];
      const crRules = spec.costReductionRules || [];

      // All test rules should be gone
      expect(noisyRules.find((r: { name: string }) => r.name?.includes('Cypress'))).to.be.undefined;
      expect(hrRules.find((r: { name: string }) => r.name?.includes('Cypress'))).to.be.undefined;
      expect(crRules.find((r: { name: string }) => r.name?.includes('Cypress'))).to.be.undefined;
    });
  });

  // ── VERIFY DELETE in UI ──────────────────────────────────────────────────

  it('Should not display any Cypress test rules in the UI after deletion', () => {
    visitPage(ROUTES.SAMPLING, () => {
      waitForGraphqlOperation('GetSamplingRules').then(() => {
        // Noisy tab (default)
        cy.contains(NOISY_RULE_NAME).should('not.exist');
        cy.contains(NOISY_RULE_NAME + UPDATED_SUFFIX).should('not.exist');

        // Highly Relevant tab
        cy.contains('button', 'Highly Relevant Operations').click();
        cy.wait(500);
        cy.contains(HIGHLY_RELEVANT_RULE_NAME).should('not.exist');
        cy.contains(HIGHLY_RELEVANT_RULE_NAME + UPDATED_SUFFIX).should('not.exist');

        // Cost Reduction tab
        cy.contains('button', 'Cost Reduction').click();
        cy.wait(500);
        cy.contains(COST_REDUCTION_RULE_NAME).should('not.exist');
        cy.contains(COST_REDUCTION_RULE_NAME + UPDATED_SUFFIX).should('not.exist');
      });
    });
  });
});
