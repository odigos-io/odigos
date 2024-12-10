import { getCrdById, getCrdIds } from '../functions';
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

    cy.contains(DATA_IDS.INSTRUMENTATION_RULE_NODE, SELECTED_ENTITIES.INSTRUMENTATION_RULE).should('exist').click();
    cy.get(DATA_IDS.DRAWER).should('exist');
    cy.get(DATA_IDS.DRAWER_EDIT).click();
    cy.get(DATA_IDS.TITLE).clear().type(TEXTS.UPDATED_NAME);
    cy.get(DATA_IDS.DRAWER_SAVE).click();
    cy.get(DATA_IDS.DRAWER_CLOSE).click();

    cy.wait('@gql').then(() => {
      getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, (crdIds) => {
        const crdId = crdIds[0];
        getCrdById({ namespace, crdName, crdId, expectedError: '', expectedKey: 'ruleName', expectedValue: TEXTS.UPDATED_NAME });
      });
    });
  });

  it('Should delete the CRD from the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    cy.contains(DATA_IDS.INSTRUMENTATION_RULE_NODE, SELECTED_ENTITIES.INSTRUMENTATION_RULE).should('exist').click();
    cy.get(DATA_IDS.DRAWER).should('exist');
    cy.get(DATA_IDS.DRAWER_EDIT).click();
    cy.get(DATA_IDS.DRAWER_DELETE).click();
    cy.get(DATA_IDS.MODAL).contains(TEXTS.INSTRUMENTATION_RULE_WARN_MODAL_TITLE).should('exist');
    cy.get(DATA_IDS.APPROVE).click();

    cy.wait('@gql').then(() => {
      getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
    });
  });
});
