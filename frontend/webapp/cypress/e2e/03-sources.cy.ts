import { getCrdById, getCrdIds, updateEntity } from '../functions';
import { BUTTONS, CRD_IDS, CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.DEFAULT;
const crdName = CRD_NAMES.SOURCE;

describe('Sources CRUD', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it('Should create a CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 }, () => {
      cy.get(DATA_IDS.ADD_ENTITY).click();
      cy.get(DATA_IDS.ADD_SOURCE).click();
      cy.get(DATA_IDS.MODAL_ADD_SOURCE).should('exist');
      cy.get(DATA_IDS.SELECT_NAMESPACE).find(DATA_IDS.CHECKBOX).click();

      // Wait for 3 seconds to allow the namespace & it's resources to be loaded into the UI
      cy.wait(3000).then(() => {
        cy.contains('button', BUTTONS.DONE).click();

        cy.wait('@gql').then(() => {
          getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 5 });
        });
      });
    });
  });

  it('Should update the CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    updateEntity(
      {
        nodeId: DATA_IDS.SOURCE_NODE,
        nodeContains: SELECTED_ENTITIES.SOURCE,
        fieldKey: DATA_IDS.SOURCE_TITLE,
        fieldValue: TEXTS.UPDATED_NAME,
      },
      () => {
        cy.wait('@gql').then(() => {
          getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 5 }, (crdIds) => {
            const crdId = CRD_IDS.SOURCE;
            expect(crdIds).includes(crdId);
            getCrdById({ namespace, crdName, crdId, expectedError: '', expectedKey: 'serviceName', expectedValue: TEXTS.UPDATED_NAME });
          });
        });
      },
    );
  });

  it('Should delete the CRD from the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    cy.get(DATA_IDS.SOURCE_NODE_HEADER).find(DATA_IDS.CHECKBOX).click();
    cy.get(DATA_IDS.MULTI_SOURCE_CONTROL).should('exist').find('button').contains(BUTTONS.UNINSTRUMENT).click();
    cy.get(DATA_IDS.MODAL).contains(TEXTS.SOURCE_WARN_MODAL_TITLE).should('exist');
    cy.get(DATA_IDS.MODAL).contains(TEXTS.SOURCE_WARN_MODAL_NOTE).should('exist');
    cy.get(DATA_IDS.APPROVE).click();

    cy.wait('@gql').then(() => {
      getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
    });
  });
});
