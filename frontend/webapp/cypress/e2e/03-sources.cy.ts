import { awaitToast, getCrdById, getCrdIds, updateEntity, visitPage } from '../functions';
import { BUTTONS, CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.DEFAULT;
const crdName = CRD_NAMES.SOURCE;
const totalEntities = SELECTED_ENTITIES.NAMESPACE_SOURCES.length;

describe('Sources CRUD', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it(`Should have 0 ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });

  it(`Should create ${totalEntities} sources via API, and notify with SSE`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      cy.get(DATA_IDS.ADD_SOURCE).click();
      cy.get(DATA_IDS.MODAL_ADD_SOURCE).should('exist');
      cy.get(DATA_IDS.SELECT_NAMESPACE).find(DATA_IDS.CHECKBOX).click({ force: true });

      // Wait for the namespace sources to load
      cy.wait('@gql').then(() => {
        SELECTED_ENTITIES.NAMESPACE_SOURCES.forEach((sourceName) => {
          cy.get(DATA_IDS.SELECT_NAMESPACE).get(DATA_IDS.SELECT_SOURCE(sourceName)).should('exist');
        });

        cy.contains('button', BUTTONS.DONE).click();

        // Wait for sources to instrument
        cy.wait('@gql').then(() => {
          awaitToast({ withSSE: true, message: TEXTS.NOTIF_SOURCES_CREATED(totalEntities) });
        });
      });
    });
  });

  it(`Should have ${totalEntities} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: totalEntities });
  });

  it(`Should update ${totalEntities} sources via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.NAMESPACE_SOURCES.forEach((sourceName, idx) => {
        updateEntity(
          {
            nodeId: DATA_IDS.SOURCE_NODE(idx),
            nodeContains: sourceName,
            fieldKey: DATA_IDS.SOURCE_TITLE,
            fieldValue: TEXTS.UPDATED_NAME,
          },
          () => {
            // Wait for the source to update
            cy.wait('@gql').then(() => {
              awaitToast({ withSSE: false, message: TEXTS.NOTIF_SOURCES_UPDATED(sourceName) });
            });
          },
        );
      });
    });
  });

  it(`Should update ${totalEntities} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: totalEntities }, (crdIds) => {
      crdIds.forEach((crdId) => {
        getCrdById({ namespace, crdName, crdId, expectedError: '', expectedKey: 'serviceName', expectedValue: TEXTS.UPDATED_NAME });
      });
    });
  });

  it(`Should delete ${totalEntities} sources via API, and notify with SSE`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      cy.get(DATA_IDS.SOURCE_NODE_HEADER).find(DATA_IDS.CHECKBOX).click();
      cy.get(DATA_IDS.MULTI_SOURCE_CONTROL).contains(totalEntities).should('exist');
      cy.get(DATA_IDS.MULTI_SOURCE_CONTROL).find('button').contains(BUTTONS.UNINSTRUMENT).click();
      cy.get(DATA_IDS.MODAL).contains(TEXTS.SOURCE_WARN_MODAL_TITLE).should('exist');
      cy.get(DATA_IDS.MODAL).contains(TEXTS.SOURCE_WARN_MODAL_NOTE).should('exist');
      cy.get(DATA_IDS.APPROVE).click();

      // Wait for the sources to delete
      cy.wait('@gql').then(() => {
        awaitToast({ withSSE: true, message: TEXTS.NOTIF_SOURCES_DELETED(totalEntities) });
      });
    });
  });

  it(`Should delete ${totalEntities} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });
});
