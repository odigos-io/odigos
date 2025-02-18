import { awaitToast, deleteEntity, getCrdById, getCrdIds, updateEntity, visitPage } from '../functions';
import { BUTTONS, CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.ODIGOS_SYSTEM;
const crdName = CRD_NAMES.DESTINATION;
const totalEntities = 1;

describe('Destinations CRUD', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it(`Should have 0 ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });

  it(`Should create ${totalEntities} ${crdName} via API, and notify with SSE`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      cy.get(DATA_IDS.ADD_DESTINATION).click();
      cy.get(DATA_IDS.MODAL_ADD_DESTINATION).should('exist');
      cy.get(DATA_IDS.SELECT_DESTINATION).contains(SELECTED_ENTITIES.DESTINATION.DISPLAY_NAME).should('exist').click();
      cy.get(DATA_IDS.SELECT_DESTINATION_AUTOFILL_FIELD).should('have.value', SELECTED_ENTITIES.DESTINATION.AUTOFILL_VALUE);
      cy.get('button').contains(BUTTONS.DONE).click();

      // Wait for destinations to create
      cy.wait('@gql').then(() => {
        awaitToast({ withSSE: true, message: TEXTS.NOTIF_DESTINATIONS_CREATED(totalEntities) });
      });
    });
  });

  it(`Should have ${totalEntities} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: totalEntities });
  });

  it(`Should update ${totalEntities} ${crdName} via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      updateEntity(
        {
          nodeId: DATA_IDS.DESTINATION_NODE(0),
          nodeContains: SELECTED_ENTITIES.DESTINATION.DISPLAY_NAME,
          fieldKey: DATA_IDS.TITLE,
          fieldValue: TEXTS.UPDATED_NAME,
        },
        () => {
          // Wait for the destination to update
          cy.wait('@gql').then(() => {
            awaitToast({ withSSE: false, message: TEXTS.NOTIF_DESTINATIONS_UPDATED(SELECTED_ENTITIES.DESTINATION.TYPE) });
          });
        },
      );
    });
  });

  it(`Should update ${totalEntities} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: totalEntities }, (crdIds) => {
      crdIds.forEach((crdId) => {
        getCrdById({ namespace, crdName, crdId, expectedError: '', expectedKey: 'destinationName', expectedValue: TEXTS.UPDATED_NAME });
      });
    });
  });

  it(`Should delete ${totalEntities} ${crdName} via API, and notify with SSE`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      deleteEntity(
        {
          nodeId: DATA_IDS.DESTINATION_NODE(0),
          nodeContains: SELECTED_ENTITIES.DESTINATION.DISPLAY_NAME,
          warnModalTitle: TEXTS.DESTINATION_WARN_MODAL_TITLE,
          warnModalNote: TEXTS.DESTINATION_WARN_MODAL_NOTE,
        },
        () => {
          // Wait for the destination to delete
          cy.wait('@gql').then(() => {
            awaitToast({ withSSE: true, message: TEXTS.NOTIF_DESTINATIONS_DELETED(totalEntities) }, () => {
              getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
            });
          });
        },
      );
    });
  });

  it(`Should delete ${totalEntities} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });
});
