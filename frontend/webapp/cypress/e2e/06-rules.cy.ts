import { MOCK_DESCRIBE_ODIGOS } from '@odigos/ui-utils';
import { BUTTONS, CRD_NAMES, DATA_IDS, INPUTS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';
import { aliasMutation, awaitToast, deleteEntity, getCrdById, getCrdIds, hasOperationName, updateEntity } from '../functions';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.ODIGOS_TEST;
const crdName = CRD_NAMES.INSTRUMENTATION_RULE;

describe('Instrumentation Rules CRUD', () => {
  beforeEach(() =>
    cy
      .intercept('/graphql', (req) => {
        aliasMutation(req, 'DescribeOdigos');

        if (hasOperationName(req, 'DescribeOdigos')) {
          req.alias = 'describeOdigos';
          req.reply((res) => {
            // This is to make the test think this is enterprise/onprem - which will allow us to create rules
            res.body.data = MOCK_DESCRIBE_ODIGOS;
          });
        }
      })
      .as('gql'),
  );

  it('Should create a CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 }, () => {
      cy.get(DATA_IDS.ADD_INSTRUMENTATION_RULE).click();
      cy.get(DATA_IDS.MODAL_ADD_INSTRUMENTATION_RULE).should('exist');
      cy.get(DATA_IDS.MODAL_ADD_INSTRUMENTATION_RULE).find('input').should('have.attr', 'placeholder', INPUTS.RULE_DROPDOWN).click();
      cy.get(DATA_IDS.RULE_DROPDOWN_OPTION).click();
      cy.get('button').contains(BUTTONS.DONE).click();

      cy.wait('@gql').then(() => {
        getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, (crdIds) => {
          const crdId = crdIds[0];
          awaitToast({ withSSE: false, message: TEXTS.NOTIF_INSTRUMENTATION_RULE_CREATED(crdId) });
        });
      });
    });
  });

  it('Should update the CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, () => {
      updateEntity(
        {
          nodeId: DATA_IDS.INSTRUMENTATION_RULE_NODE,
          nodeContains: SELECTED_ENTITIES.INSTRUMENTATION_RULE,
          fieldKey: DATA_IDS.TITLE,
          fieldValue: TEXTS.UPDATED_NAME,
        },
        () => {
          cy.wait('@gql').then(() => {
            getCrdIds({ namespace, crdName, expectedError: '', expectedLength: 1 }, (crdIds) => {
              const crdId = crdIds[0];
              awaitToast({ withSSE: false, message: TEXTS.NOTIF_INSTRUMENTATION_RULE_UPDATED(crdId) }, () => {
                getCrdById({ namespace, crdName, crdId, expectedError: '', expectedKey: 'ruleName', expectedValue: TEXTS.UPDATED_NAME });
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
          nodeId: DATA_IDS.INSTRUMENTATION_RULE_NODE,
          nodeContains: SELECTED_ENTITIES.INSTRUMENTATION_RULE,
          warnModalTitle: TEXTS.INSTRUMENTATION_RULE_WARN_MODAL_TITLE,
        },
        () => {
          cy.wait('@gql').then(() => {
            awaitToast({ withSSE: false, message: TEXTS.NOTIF_INSTRUMENTATION_RULE_DELETED(crdId) }, () => {
              getCrdIds({ namespace, crdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
            });
          });
        },
      );
    });
  });
});
