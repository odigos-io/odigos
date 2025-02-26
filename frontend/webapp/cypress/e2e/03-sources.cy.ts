import { awaitToast, getCrdById, getCrdIds, updateEntity, visitPage } from '../functions';
import { BUTTONS, CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.DEFAULT;
const sourceCrdName = CRD_NAMES.SOURCE;
const configCrdName = CRD_NAMES.INSTRUMENTATION_CONFIG;
const totalEntities = SELECTED_ENTITIES.NAMESPACE_SOURCES.length;

describe('Sources CRUD', () => {
  beforeEach(() => {
    cy.intercept('/graphql').as('gql');

    cy.on('uncaught:exception', (err, runnable) => {
      if (err.message.includes('ResizeObserver loop completed with undelivered notifications')) {
        // returning false here prevents Cypress from failing the test
        return false;
      }
      // we still want to ensure there are no other unexpected errors, so we let them fail the test
    });
  });

  it(`Should have 0 ${sourceCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: sourceCrdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });

  it(`Should have 0 ${configCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: configCrdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });

  it(`Should create ${totalEntities} sources via API, and notify with SSE`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      cy.get(DATA_IDS.ADD_SOURCE).click();
      cy.get(DATA_IDS.MODAL_ADD_SOURCE).should('exist');
      cy.get(DATA_IDS.SELECT_NAMESPACE).find(DATA_IDS.CHECKBOX).click();

      // Wait for the namespace sources to load
      cy.wait(1000).then(() => {
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

  it(`Should have ${totalEntities + 1} ${sourceCrdName} CRDs in the cluster (with +1 ${sourceCrdName} CRD for namespace)`, () => {
    getCrdIds({ namespace, crdName: sourceCrdName, expectedError: '', expectedLength: totalEntities + 1 });
  });

  it(`Should have ${totalEntities} ${configCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: configCrdName, expectedError: '', expectedLength: totalEntities });
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

              // Since we're updating all sources, and the modified event batcher (in SSE) refreshes the sources...
              // We will force an extra 3 seconds-wait before we continue to the next source in the loop, this is to ensure we have an updated UI before we proceed to update the next source (otherwise Cypress will fail to find the elements).
              cy.wait(3000).then(() => {
                expect(true).to.be.true;
              });
            });
          },
        );
      });
    });
  });

  it(`Should update ${totalEntities + 1} ${sourceCrdName} CRDs in the cluster (with +1 ${sourceCrdName} CRD for namespace)`, () => {
    getCrdIds({ namespace, crdName: sourceCrdName, expectedError: '', expectedLength: totalEntities + 1 }, (crdIds) => {
      crdIds.forEach((crdId) => {
        getCrdById({ namespace, crdName: sourceCrdName, crdId, expectedError: '', expectedKey: 'serviceName', expectedValue: TEXTS.UPDATED_NAME });
      });
    });
  });

  it(`Should update ${totalEntities} ${configCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: configCrdName, expectedError: '', expectedLength: totalEntities }, (crdIds) => {
      crdIds.forEach((crdId) => {
        getCrdById({ namespace, crdName: configCrdName, crdId, expectedError: '', expectedKey: 'serviceName', expectedValue: TEXTS.UPDATED_NAME });
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

  it(`Should update ${totalEntities} ${sourceCrdName} CRDs in the cluster with "disabled=true" (and skip +1 ${sourceCrdName} CRD for namespace)`, () => {
    getCrdIds({ namespace, crdName: sourceCrdName, expectedError: '', expectedLength: totalEntities + 1 }, (crdIds) => {
      crdIds.forEach((crdId) => {
        getCrdById({ namespace, crdName: sourceCrdName, crdId, expectedError: '', expectedKey: 'disableInstrumentation', expectedValue: true });
      });
    });
  });

  it(`Should have ${0} ${configCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: configCrdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });
});
