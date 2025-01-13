import { awaitToast, deleteEntity, getCrdById, getCrdIds, updateEntity } from '../functions';
import { BUTTONS, CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.ODIGOS_SYSTEM;
const crdName = CRD_NAMES.DESTINATION;

describe('Destinations CRUD', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it('Should create a CRD in the cluster, and notify with SSE', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 }, () => {
      cy.get(DATA_IDS.ADD_ENTITY).click();
      cy.get(DATA_IDS.ADD_DESTINATION).click();
      cy.get(DATA_IDS.MODAL_ADD_DESTINATION).should('exist');
      cy.get(DATA_IDS.SELECT_DESTINATION).contains(SELECTED_ENTITIES.DESTINATION_DISPLAY_NAME).should('exist').click();
      cy.get(DATA_IDS.SELECT_DESTINATION_AUTOFILL_FIELD).should('have.value', SELECTED_ENTITIES.DESTINATION_AUTOFILL_VALUE);
      cy.get('button').contains(BUTTONS.DONE).click();

      cy.wait('@gql').then(() => {
        awaitToast({ withSSE: true, message: TEXTS.NOTIF_DESTINATIONS_CREATED(1) }, () => {
          getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 });
        });
      });
    });
  });

  it('Should update the CRD in the cluster, and notify with SSE', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, () => {
      updateEntity(
        {
          nodeId: DATA_IDS.DESTINATION_NODE,
          nodeContains: SELECTED_ENTITIES.DESTINATION_DISPLAY_NAME,
          fieldKey: DATA_IDS.TITLE,
          fieldValue: TEXTS.UPDATED_NAME,
        },
        () => {
          cy.wait('@gql').then(() => {
            getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, (crdIds) => {
              const crdId = crdIds[0];

              awaitToast({ withSSE: true, message: TEXTS.NOTIF_DESTINATIONS_UPDATED(1) }, () => {
                getCrdById({ namespace, crdName, crdId, expectedError: '', expectedKey: 'destinationName', expectedValue: TEXTS.UPDATED_NAME });
              });
            });
          });
        },
      );
    });
  });

  it('Should delete the CRD from the cluster, and notify with SSE', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, () => {
      deleteEntity(
        {
          nodeId: DATA_IDS.DESTINATION_NODE,
          nodeContains: SELECTED_ENTITIES.DESTINATION_DISPLAY_NAME,
          warnModalTitle: TEXTS.DESTINATION_WARN_MODAL_TITLE,
          warnModalNote: TEXTS.DESTINATION_WARN_MODAL_NOTE,
        },
        () => {
          cy.wait('@gql').then(() => {
            awaitToast({ withSSE: true, message: TEXTS.NOTIF_DESTINATIONS_DELETED(1) }, () => {
              getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
            });
          });
        },
      );
    });
  });
});
