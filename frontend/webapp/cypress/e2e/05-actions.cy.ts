import { CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';
import { awaitToast, deleteV2Entity, getCrdById, getCrdIds, handleExceptions, updateV2Entity, visitPage, waitForGraphqlOperation } from '../functions';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.ODIGOS;
const crdName = CRD_NAMES.ACTION;
const totalEntities = SELECTED_ENTITIES.ACTIONS.length;

describe('Actions CRUD', () => {
  beforeEach(() => {
    cy.intercept('/graphql').as('gql');
    handleExceptions();
  });

  it(`Should have 0 ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });

  it(`Should create ${totalEntities} actions via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.ACTIONS.forEach((actionType) => {
        cy.get(DATA_IDS.ADD_ACTION).click();

        // Select action type from the drawer's left column list
        cy.get(DATA_IDS.ACTION_OPTION(actionType)).should('exist').click();

        switch (actionType) {
          case 'K8sAttributesResolver': {
            // The dynamic (catalog-driven) form no longer force-enables the
            // collect flags on mount like the old bespoke form did, so the
            // action would validate as a no-op ("Enable at least one option").
            // Toggle one flag on to make it a valid action.
            cy.get('[data-id=collectContainerAttributes]').click();
            break;
          }
          case 'AddClusterInfo': {
            // The dynamic table cells expose the column keyName as the input's
            // data-id (see ui-kit InputTable -> Input), so target those instead
            // of the (now catalog-defined) placeholder text.
            cy.get('[data-id=clusterAttributes]').find('input[data-id=attributeName]').type('key');
            cy.get('[data-id=clusterAttributes]').find('input[data-id=attributeStringValue]').type('val');
            break;
          }
          case 'DeleteAttribute': {
            cy.get('[data-id=attributeNamesToDelete]').find('input').first().type('test');
            break;
          }
          // NOTE: 'RenameAttribute' is intentionally omitted from SELECTED_ENTITIES.ACTIONS
          // because it can't be created via the dynamic form yet (PLAT-1260).
          case 'PiiMasking': {
            // The dynamic form renders piiCategories as a free-text multiInput with
            // no default (the old bespoke form pre-selected CREDIT_CARD), so enter a
            // valid category to produce a non-empty action.
            cy.get('[data-id=piiCategories]').find('input').first().type('CREDIT_CARD');
            break;
          }
          // NOTE: 'ExtractAttribute' is intentionally omitted from SELECTED_ENTITIES.ACTIONS
          // because the dynamic form sends `dataFormat: ""` for an unselected optional
          // enum, which the GraphQL server rejects with a 400 (PLAT-1261). Restore this
          // case once that's fixed:
          //
          // case 'ExtractAttribute': {
          //   cy.get('[data-id=extractAttribute]').find('input[data-id=targetAttributeName]').type('extracted.value');
          //   cy.get('[data-id=extractAttribute]').find('input[data-id=regex]').type('(.*)');
          //   break;
          // }
          default: {
            // purposely fail the test
            cy.get('unknown action').should('eq', true);
            break;
          }
        }

        cy.get(DATA_IDS.WIDE_DRAWER_SAVE).click();

        // Wait for action to create
        waitForGraphqlOperation('CreateAction').then(() => {
          awaitToast({ message: TEXTS.NOTIF_ACTION_CREATED(actionType) });
        });
      });
    });
  });

  it(`Should have ${totalEntities} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: totalEntities });
  });

  it(`Should update ${totalEntities} actions via the v2 edit-action-drawer, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.ACTIONS.forEach((actionType) => {
        updateV2Entity(
          {
            // actions are fetched in random order, so we locate the row by the type text it shows
            nodeId: 'div',
            nodeContains: actionType,
            prefix: DATA_IDS.ACTION_DRAWER_PREFIX,
            fieldKey: DATA_IDS.ACTION_NAME_INPUT,
            // Embed the action type in the new name. The v2 ListItem renders
            // `name || type`, so renaming every row to a single shared value
            // would erase the per-row text we use to find rows in the delete
            // test (`cy.contains('div', actionType)`). Keeping `actionType` in
            // the value lets that substring lookup keep working.
            fieldValue: `${TEXTS.UPDATED_NAME} ${actionType}`,
          },
          () => {
            // Wait for the action to update
            waitForGraphqlOperation('UpdateAction').then(() => {
              awaitToast({ message: TEXTS.NOTIF_ACTION_UPDATED(actionType) });
            });
          },
        );
      });
    });
  });

  it(`Should update ${totalEntities} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: totalEntities }, (crdIds) => {
      crdIds.forEach((crdId) => {
        // Each action's `actionName` was renamed to `${UPDATED_NAME} ${type}`;
        // verify the shared marker landed on every CRD without coupling to the
        // CRD-id ↔ actionType mapping (which we don't track here).
        getCrdById({ namespace, crdName, crdId, expectedError: '', expectedKey: 'actionName', expectedValue: TEXTS.UPDATED_NAME, expectedValueContains: true });
      });
    });
  });

  it(`Should delete ${totalEntities} actions via the v2 edit-action-drawer, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.ACTIONS.forEach((actionType) => {
        deleteV2Entity(
          {
            nodeId: 'div',
            nodeContains: actionType,
            prefix: DATA_IDS.ACTION_DRAWER_PREFIX,
            warnModalTitle: TEXTS.ACTION_WARN_MODAL_TITLE,
          },
          () => {
            // Wait for the action to delete
            waitForGraphqlOperation('DeleteAction').then(() => {
              awaitToast({ message: TEXTS.NOTIF_ACTION_DELETED(actionType) });
            });
          },
        );
      });
    });
  });

  it(`Should have ${0} ${crdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });
});
