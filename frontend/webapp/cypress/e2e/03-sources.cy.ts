import { BUTTONS, CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';
import { awaitToast, getCrdById, getCrdIds, handleExceptions, updateEntity, visitPage } from '../functions';

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
    handleExceptions();
  });

  it(`Should have 0 ${sourceCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: sourceCrdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });

  it(`Should have 0 ${configCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: configCrdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });

  it(`Should instrument ${totalEntities} sources via API, and notify with SSE`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      cy.get(DATA_IDS.ADD_SOURCE).click();
      cy.get(DATA_IDS.MODAL_ADD_SOURCE).should('exist');
      cy.get(DATA_IDS.SELECT_NAMESPACE).find(DATA_IDS.CHECKBOX).click();

      // Wait for the namespace sources to load
      cy.wait(500).then(() => {
        SELECTED_ENTITIES.NAMESPACE_SOURCES.forEach((sourceName) => {
          cy.get(DATA_IDS.SELECT_NAMESPACE).get(DATA_IDS.SELECT_SOURCE(sourceName)).should('exist');
        });

        cy.contains('button', BUTTONS.DONE).click();

        awaitToast({ message: TEXTS.NOTIF_SOURCES_PERSISTING });
        // Wait for sources to instrument
        cy.wait('@gql').then(() => {
          awaitToast({ message: TEXTS.NOTIF_SOURCES_CREATED(totalEntities) });
        });
      });
    });
  });

  it(`Should have ${totalEntities} ${sourceCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: sourceCrdName, expectedError: '', expectedLength: totalEntities });
  });

  it(`Should have ${totalEntities} ${configCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: configCrdName, expectedError: '', expectedLength: totalEntities });
  });

  it(`Should update the name of ${totalEntities} sources via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.NAMESPACE_SOURCES.forEach((sourceName, idx) => {
        updateEntity(
          {
            nodeId: DATA_IDS.SOURCE_NODE(idx),
            fieldKey: DATA_IDS.SOURCE_TITLE,
            fieldValue: TEXTS.UPDATED_NAME,
          },
          () => {
            awaitToast({ message: TEXTS.NOTIF_SOURCE_UPDATING });
            // Wait for the source to update
            cy.wait('@gql').then(() => {
              awaitToast({ message: TEXTS.NOTIF_UPDATED });
              // Wait for the cluster to inherit the changes...
              cy.wait(500).then(() => expect(true).to.be.true);
            });
          },
        );
      });
    });
  });

  it(`Should update "otelServiceName" of ${totalEntities} ${sourceCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: sourceCrdName, expectedError: '', expectedLength: totalEntities }, (crdIds) => {
      crdIds.forEach((crdId) => {
        getCrdById({ namespace, crdName: sourceCrdName, crdId, expectedError: '', expectedKey: 'otelServiceName', expectedValue: TEXTS.UPDATED_NAME });
      });
    });
  });

  it(`Should update "serviceName" of ${totalEntities} ${configCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: configCrdName, expectedError: '', expectedLength: totalEntities }, (crdIds) => {
      crdIds.forEach((crdId) => {
        getCrdById({ namespace, crdName: configCrdName, crdId, expectedError: '', expectedKey: 'serviceName', expectedValue: TEXTS.UPDATED_NAME });
      });
    });
  });

  it(`Should uninstrument ${totalEntities} sources via API, and notify with SSE`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      cy.get(DATA_IDS.SOURCE_NODE_HEADER).find(DATA_IDS.CHECKBOX).click();
      cy.get(DATA_IDS.MULTI_SOURCE_CONTROL).contains(totalEntities).should('exist');
      cy.get(DATA_IDS.MULTI_SOURCE_CONTROL).find('button').contains(BUTTONS.UNINSTRUMENT).click();
      cy.get(DATA_IDS.MODAL).contains(TEXTS.SOURCE_WARN_MODAL_TITLE(totalEntities)).should('exist');
      cy.get(DATA_IDS.MODAL).contains(TEXTS.SOURCE_WARN_MODAL_NOTE).should('exist');
      cy.get(DATA_IDS.APPROVE).click();

      awaitToast({ message: TEXTS.NOTIF_SOURCES_PERSISTING });
      // Wait for the sources to delete
      cy.wait('@gql').then(() => {
        awaitToast({ message: TEXTS.NOTIF_SOURCES_DELETED(totalEntities) });
      });
    });
  });

  it(`Should have 0 ${sourceCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: sourceCrdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });

  it(`Should have 0 ${configCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: configCrdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });
});
