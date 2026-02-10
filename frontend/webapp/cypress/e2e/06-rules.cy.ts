import { BUTTONS, CRD_NAMES, DATA_IDS, INPUTS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';
import { aliasQuery, awaitToast, deleteEntity, getCrdById, getCrdIds, handleExceptions, hasOperationName, updateEntity, visitPage } from '../functions';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.ODIGOS_TEST;
const crdName = CRD_NAMES.INSTRUMENTATION_RULE;
const totalEntities = SELECTED_ENTITIES.INSTRUMENTATION_RULES.length;

describe('Instrumentation Rules CRUD', () => {
  beforeEach(() => {
    cy.intercept('/graphql', (req) => {
      if (hasOperationName(req, 'GetConfig')) {
        aliasQuery(req, 'GetConfig');

        req.reply((res) => {
          // This is to make the test think this is enterprise/onprem - which will allow us to create rules
          res.body.data = { config: { tier: 'onprem' } };
        });
      }
    }).as('gql');

    handleExceptions();
  });

  it(`Should have 0 ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });

  it(`Should create ${totalEntities} rules via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.INSTRUMENTATION_RULES.forEach((ruleType) => {
        cy.get(DATA_IDS.ADD_INSTRUMENTATION_RULE).click();
        cy.get(DATA_IDS.MODAL_ADD_INSTRUMENTATION_RULE).should('exist');
        cy.get(DATA_IDS.MODAL_ADD_INSTRUMENTATION_RULE).find('input').should('have.attr', 'placeholder', INPUTS.RULE_DROPDOWN).click();
        cy.get(DATA_IDS.RULE_OPTION(ruleType)).click();
        // No need to fill form (as we did in actions), because default values are enough 👍 for all rules
        cy.get('button').contains(BUTTONS.DONE).click();

        // Wait for rule to create
        cy.wait('@gql').then(() => {
          awaitToast({ message: TEXTS.NOTIF_INSTRUMENTATION_RULE_CREATED(ruleType) });
        });
      });
    });
  });

  it(`Should have ${totalEntities} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: totalEntities });
  });

  it(`Should update ${totalEntities} rules via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.INSTRUMENTATION_RULES.forEach((ruleType) => {
        updateEntity(
          {
            // no indexed node, because rules are fetched in random order
            nodeId: 'div',
            nodeContains: ruleType,
            fieldKey: DATA_IDS.TITLE,
            fieldValue: TEXTS.UPDATED_NAME,
          },
          () => {
            // Wait for the rule to update
            cy.wait('@gql').then(() => {
              awaitToast({ message: TEXTS.NOTIF_INSTRUMENTATION_RULE_UPDATED(ruleType) });
            });
          },
        );
      });
    });
  });

  it(`Should update ${totalEntities} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: totalEntities }, (crdIds) => {
      crdIds.forEach((crdId) => {
        getCrdById({ namespace, crdName, crdId, expectedError: '', expectedKey: 'ruleName', expectedValue: TEXTS.UPDATED_NAME });
      });
    });
  });

  it(`Should delete ${totalEntities} actions via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.INSTRUMENTATION_RULES.forEach((ruleType) => {
        deleteEntity(
          {
            // no indexed node, because rules are fetched in random order
            nodeId: 'div',
            nodeContains: ruleType,
            warnModalTitle: TEXTS.INSTRUMENTATION_RULE_WARN_MODAL_TITLE,
          },
          () => {
            // Wait for the rule to delete
            cy.wait('@gql').then(() => {
              awaitToast({ message: TEXTS.NOTIF_INSTRUMENTATION_RULE_DELETED(ruleType) });
            });
          },
        );
      });
    });
  });

  it(`Should have ${0} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });
});
