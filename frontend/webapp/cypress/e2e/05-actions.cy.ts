import { awaitToast, deleteEntity, getCrdById, getCrdIds, updateEntity } from '../functions';
import { BUTTONS, CRD_NAMES, DATA_IDS, INPUTS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.ODIGOS_SYSTEM;
const crdName = CRD_NAMES.ACTION;

describe('Actions CRUD', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it('Should create a CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 }, () => {
      cy.get(DATA_IDS.ADD_ENTITY).click();
      cy.get(DATA_IDS.ADD_ACTION).click();
      cy.get(DATA_IDS.MODAL_ADD_ACTION).should('exist');
      cy.get(DATA_IDS.MODAL_ADD_ACTION).find('input').should('have.attr', 'placeholder', INPUTS.ACTION_DROPDOWN).click();
      cy.get(DATA_IDS.ACTION_DROPDOWN_OPTION).click();
      cy.get('button').contains(BUTTONS.DONE).click();

      cy.wait('@gql').then(() => {
        getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, (crdIds) => {
          const crdId = crdIds[0];
          awaitToast({ withSSE: false, message: TEXTS.NOTIF_ACTION_CREATED(crdId) });
        });
      });
    });
  });

  it('Should update the CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, () => {
      updateEntity(
        {
          nodeId: DATA_IDS.ACTION_NODE,
          nodeContains: SELECTED_ENTITIES.ACTION,
          fieldKey: DATA_IDS.TITLE,
          fieldValue: TEXTS.UPDATED_NAME,
        },
        () => {
          cy.wait('@gql').then(() => {
            getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, (crdIds) => {
              const crdId = crdIds[0];
              awaitToast({ withSSE: false, message: TEXTS.NOTIF_ACTION_UPDATED(crdId) }, () => {
                getCrdById({ namespace, crdName, crdId, expectedError: '', expectedKey: 'actionName', expectedValue: TEXTS.UPDATED_NAME });
              });
            });
          });
        },
      );
    });
  });

  it('Should delete the CRD from the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, (crdIds) => {
      const crdId = crdIds[0];
      deleteEntity(
        {
          nodeId: DATA_IDS.ACTION_NODE,
          nodeContains: SELECTED_ENTITIES.ACTION,
          warnModalTitle: TEXTS.ACTION_WARN_MODAL_TITLE,
        },
        () => {
          cy.wait('@gql').then(() => {
            awaitToast({ withSSE: false, message: TEXTS.NOTIF_ACTION_DELETED(crdId) }, () => {
              getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
            });
          });
        },
      );
    });
  });
});
