import { deleteEntity, getCrdById, getCrdIds, updateEntity } from '../functions';
import { BUTTONS, CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.ODIGOS_SYSTEM;
const crdName = CRD_NAMES.INSTRUMENTATION_RULE;

describe('Instrumentation Rules CRUD', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it('Should create a CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 }, () => {
      cy.get(DATA_IDS.ADD_ENTITY).click();
      cy.get(DATA_IDS.ADD_INSTRUMENTATION_RULE).click();
      cy.get(DATA_IDS.MODAL_ADD_INSTRUMENTATION_RULE).should('exist');
      cy.get('button').contains(BUTTONS.DONE).click();

      cy.wait('@gql').then(() => {
        getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 });
      });
    });
  });

  it('Should update the CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, () => {
      updateEntity(
        {
          nodeId: DATA_IDS.INSTRUMENTATION_RULE_NODE,
          nodeContains: SELECTED_ENTITIES.INSTRUMENTATION_RULE,
          fieldKey: DATA_IDS.TITLE,
          fieldValue: TEXTS.UPDATED_NAME,
        },
        () => {
          cy.wait('@gql').then(() => {
            getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, (crdIds) => {
              const crdId = crdIds[0];
              getCrdById({ namespace, crdName, crdId, expectedError: '', expectedKey: 'ruleName', expectedValue: TEXTS.UPDATED_NAME });
            });
          });
        },
      );
    });
  });

  it('Should delete the CRD from the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, () => {
      deleteEntity(
        {
          nodeId: DATA_IDS.INSTRUMENTATION_RULE_NODE,
          nodeContains: SELECTED_ENTITIES.INSTRUMENTATION_RULE,
          warnModalTitle: TEXTS.INSTRUMENTATION_RULE_WARN_MODAL_TITLE,
        },
        () => {
          cy.wait('@gql').then(() => {
            getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
          });
        },
      );
    });
  });
});
