import { awaitToast, deleteEntity, getCrdById, getCrdIds, updateEntity, visitPage } from '../functions';
import { BUTTONS, CRD_NAMES, DATA_IDS, INPUTS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.ODIGOS_SYSTEM;
const crdNames = CRD_NAMES.ACTIONS;
const totalEntities = SELECTED_ENTITIES.ACTIONS.length;

describe('Actions CRUD', () => {
  beforeEach(() => {
    cy.intercept('/graphql').as('gql');

    cy.on('uncaught:exception', (err, runnable) => {
      if (err.message.includes('ResizeObserver loop completed with undelivered notifications')) {
        // returning false here prevents Cypress from failing the test
        return false;
      }

      return true;
    });
  });

  it(`Should have 0 ${JSON.stringify(crdNames)} CRDs in the cluster`, () => {
    expect(crdNames.length).to.eq(totalEntities);
    crdNames.forEach((crdName) => {
      getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
    });
  });

  it(`Should create ${totalEntities} actions via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.ACTIONS.forEach((actionType) => {
        cy.get(DATA_IDS.ADD_ACTION).click();
        cy.get(DATA_IDS.MODAL_ADD_ACTION).should('exist');
        cy.get(DATA_IDS.MODAL_ADD_ACTION).find('input').should('have.attr', 'placeholder', INPUTS.ACTION_DROPDOWN).click();
        cy.get(DATA_IDS.ACTION_OPTION(actionType)).click();

        switch (actionType) {
          case 'K8sAttributesResolver': {
            // default values are enough 👍
            break;
          }
          case 'AddClusterInfo': {
            cy.contains('div', 'Resource Attributes').parent().parent().find('input[placeholder="Attribute name"]').type('key');
            cy.contains('div', 'Resource Attributes').parent().parent().find('input[placeholder="Attribute value"]').type('val');
            break;
          }
          case 'DeleteAttribute': {
            cy.contains('div', 'Attributes to delete').parent().parent().find('input').type('test');
            break;
          }
          case 'RenameAttribute': {
            cy.contains('div', 'Attributes to rename').parent().parent().find('input[placeholder="Attribute name"]').type('1');
            cy.contains('div', 'Attributes to rename').parent().parent().find('input[placeholder="Attribute value"]').type('one');
            break;
          }
          case 'PiiMasking': {
            // default values are enough 👍
            break;
          }
          case 'ErrorSampler': {
            cy.contains('div', 'Fallback sampling ratio').parent().parent().find('input').type('1');
            break;
          }
          case 'LatencySampler': {
            cy.get('tbody').find('input[placeholder="Choose service"]').type('service');
            cy.get('tbody').find('input[placeholder="e.g. /api/v1/users"]').type('/path');
            cy.get('tbody').find('input[placeholder="e.g. 1000"]').type('1');
            cy.get('tbody').find('input[placeholder="e.g. 20"]').type('1');
            break;
          }
          case 'ProbabilisticSampler': {
            cy.contains('div', 'Sampling percentage').parent().parent().find('input').type('1');
            break;
          }

          default: {
            // purposely fail the test
            cy.get('unknown action').should('eq', true);
            break;
          }
        }

        cy.get('button').contains(BUTTONS.DONE).click();

        // Wait for action to create
        cy.wait('@gql').then(() => {
          awaitToast({ withSSE: false, message: TEXTS.NOTIF_ACTION_CREATED(actionType) });
        });
      });
    });
  });

  it(`Should have ${totalEntities} ${JSON.stringify(crdNames)} CRDs in the cluster`, () => {
    expect(crdNames.length).to.eq(totalEntities);
    crdNames.forEach((crdName) => {
      // always 1, because each CRD has a unique name (actions only)
      getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 });
    });
  });

  it(`Should update ${totalEntities} actions via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.ACTIONS.forEach((actionType, idx) => {
        updateEntity(
          {
            nodeId: DATA_IDS.ACTION_NODE(idx),
            nodeContains: actionType,
            fieldKey: DATA_IDS.TITLE,
            fieldValue: TEXTS.UPDATED_NAME,
          },
          () => {
            // Wait for the action to update
            cy.wait('@gql').then(() => {
              awaitToast({ withSSE: false, message: TEXTS.NOTIF_ACTION_UPDATED(actionType) });
            });
          },
        );
      });
    });
  });

  it(`Should update ${totalEntities} ${JSON.stringify(crdNames)} CRDs in the cluster`, () => {
    expect(crdNames.length).to.eq(totalEntities);
    crdNames.forEach((crdName) => {
      // always 1, because each CRD has a unique name (actions only)
      getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, (crdIds) => {
        crdIds.forEach((crdId) => {
          getCrdById({ namespace, crdName, crdId, expectedError: '', expectedKey: 'actionName', expectedValue: TEXTS.UPDATED_NAME });
        });
      });
    });
  });

  it(`Should delete ${totalEntities} actions via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.ACTIONS.forEach((actionType) => {
        deleteEntity(
          {
            // always index 0, because when one action is deleted, it shifts the index of the next action
            nodeId: DATA_IDS.ACTION_NODE(0),
            nodeContains: actionType,
            warnModalTitle: TEXTS.ACTION_WARN_MODAL_TITLE,
          },
          () => {
            // Wait for the action to delete
            cy.wait('@gql').then(() => {
              awaitToast({ withSSE: false, message: TEXTS.NOTIF_ACTION_DELETED(actionType) });
            });
          },
        );
      });
    });
  });

  it(`Should delete ${totalEntities} ${JSON.stringify(crdNames)} CRDs in the cluster`, () => {
    expect(crdNames.length).to.eq(totalEntities);
    crdNames.forEach((crdName) => {
      getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
    });
  });
});
