import { CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';
import { awaitToast, deleteEntity, getCrdById, getCrdIds, handleExceptions, updateEntity, visitPage } from '../functions';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.ODIGOS;
const crdName = CRD_NAMES.DESTINATION;
const totalEntities = 1;
const destinationIds: string[] = [];

describe('Destinations CRUD', () => {
  beforeEach(() => {
    cy.intercept('/graphql').as('gql');
    handleExceptions();
  });

  it(`Should have 0 ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });

  it(`Should create ${totalEntities} destinations via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      cy.get(DATA_IDS.ADD_DESTINATION).click();

      // Select destination from the drawer's list (force needed for virtualized lists)
      cy.get(DATA_IDS.SELECT_DESTINATION).first().should('exist').click({ force: true });
      cy.get(DATA_IDS.SELECT_DESTINATION_AUTOFILL_FIELD).should('have.value', SELECTED_ENTITIES.DESTINATION.AUTOFILL_VALUE);

      // Add destination to unsaved list, then save
      cy.get(DATA_IDS.DEST_FORM_ADD).click();
      cy.get(DATA_IDS.WIDE_DRAWER_SAVE).click();

      // Wait for the GraphQL mutation and the drawer to close
      cy.wait('@gql').then(() => {
        cy.get(DATA_IDS.WIDE_DRAWER_SAVE).should('not.exist');
        cy.wait(2000);
      });
    });
  });

  it(`Should have ${totalEntities} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: totalEntities }, (crdIds) => {
      crdIds.forEach((crdId) => {
        destinationIds.push(crdId);
      });
    });
  });

  it(`Should update ${totalEntities} destinations via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      updateEntity(
        {
          nodeId: DATA_IDS.DESTINATION_NODE(destinationIds[0]),
          nodeContains: SELECTED_ENTITIES.DESTINATION.DISPLAY_NAME,
          fieldKey: DATA_IDS.TITLE,
          fieldValue: TEXTS.UPDATED_NAME,
        },
        () => {
          // Wait for the destination to update
          cy.wait('@gql').then(() => {
            awaitToast({ message: TEXTS.NOTIF_DESTINATION_UPDATED(SELECTED_ENTITIES.DESTINATION.TYPE) });
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

  it(`Should delete ${totalEntities} destinations via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      deleteEntity(
        {
          nodeId: DATA_IDS.DESTINATION_NODE(destinationIds[0]),
          nodeContains: SELECTED_ENTITIES.DESTINATION.DISPLAY_NAME,
          warnModalTitle: TEXTS.DESTINATION_WARN_MODAL_TITLE,
          warnModalNote: TEXTS.DESTINATION_WARN_MODAL_NOTE,
        },
        () => {
          // Wait for the destination to delete
          cy.wait('@gql').then(() => {
            awaitToast({ message: TEXTS.NOTIF_DESTINATION_DELETED(totalEntities) }, () => {
              getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
            });
          });
        },
      );
    });
  });

  it(`Should have ${0} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });
});
