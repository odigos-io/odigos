import { BUTTONS, CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';
import { awaitToast, findCrdId, getCrdById, getCrdIds, handleExceptions, updateEntity, visitPage } from '../functions';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

const namespace = NAMESPACES.APPS;
const sourceCrdName = CRD_NAMES.SOURCE;
const configCrdName = CRD_NAMES.INSTRUMENTATION_CONFIG;
const totalEntities = SELECTED_ENTITIES.NAMESPACE_SOURCES.length;
const indexForUpdatedSource = 0;
const nameForUpdatedSource = SELECTED_ENTITIES.NAMESPACE_SOURCES[indexForUpdatedSource].name;

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

  it(`Should instrument ${totalEntities} sources via API`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      cy.get(DATA_IDS.ADD_SOURCE).click();

      // Wait for the drawer to load namespaces, then click to activate (show workloads)
      cy.get(DATA_IDS.SELECT_NAMESPACE).should('exist').click();

      // Select all workloads individually via the workloads column "Select all"
      cy.wait(500).then(() => {
        cy.get('[data-id=workloads-select-all]').click();

        SELECTED_ENTITIES.NAMESPACE_SOURCES.forEach(({ name }) => {
          cy.get(DATA_IDS.SELECT_SOURCE(name)).should('exist');
        });

        cy.get(DATA_IDS.WIDE_DRAWER_SAVE).click();

        // Wait for the drawer to close (v2 closes on successful save)
        cy.get(DATA_IDS.WIDE_DRAWER_SAVE).should('not.exist');
      });
    });
  });

  it(`Should have 1 ${sourceCrdName} CRD in the cluster (namespace-level)`, () => {
    getCrdIds({ namespace, crdName: sourceCrdName, expectedError: '', expectedLength: 1 });
  });

  it(`Should have >= ${totalEntities} ${configCrdName} CRDs in the cluster`, () => {
    cy.exec(`kubectl get ${configCrdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then(({ stdout }) => {
      const count = stdout.split('\n').filter((s) => !!s).length;
      expect(count).to.be.gte(totalEntities);
    });
  });

  // Note: we update only 1 source, because Cypress keeps flaking when updating all of them.
  it(`Should update the name of ${1} sources via API, and notify locally`, () => {
    visitPage(ROUTES.OVERVIEW, () => {
      SELECTED_ENTITIES.NAMESPACE_SOURCES.slice(indexForUpdatedSource, indexForUpdatedSource + 1).forEach(({ name, kind }) => {
        updateEntity(
          {
            nodeId: DATA_IDS.SOURCE_NODE({ namespace, name, kind }),
            fieldKey: DATA_IDS.SOURCE_TITLE,
            fieldValue: TEXTS.UPDATED_NAME,
          },
          () => {
            awaitToast({ message: TEXTS.NOTIF_SOURCE_UPDATING });
            // Wait for the source to update
            cy.wait('@gql').then(() => {
              awaitToast({ message: TEXTS.NOTIF_UPDATED });
            });
          },
        );
      });
    });
  });

  // Note: we update only 1 source, because Cypress keeps flaking when updating all of them.
  it(`Should update "otelServiceName" of ${1} ${sourceCrdName} CRDs in the cluster`, () => {
    findCrdId({ namespace, crdName: sourceCrdName, targetKey: 'workload.name', targetValue: nameForUpdatedSource }, (crdId) => {
      getCrdById({ namespace, crdName: sourceCrdName, crdId, expectedError: '', expectedKey: 'otelServiceName', expectedValue: TEXTS.UPDATED_NAME });
    });
  });

  // Note: we update only 1 source, because Cypress keeps flaking when updating all of them.
  it(`Should update "serviceName" of ${1} ${configCrdName} CRDs in the cluster`, () => {
    cy.exec(`kubectl get ${configCrdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then(({ stdout }) => {
      const crdIds = stdout.split('\n').filter((s) => !!s);
      expect(crdIds.length).to.be.gte(totalEntities);

      const crdId = crdIds.find((id) => id.indexOf(nameForUpdatedSource) !== -1) || '';
      getCrdById({ namespace, crdName: configCrdName, crdId, expectedError: '', expectedKey: 'serviceName', expectedValue: TEXTS.UPDATED_NAME });
    });
  });

  it('Should uninstrument all sources via API', () => {
    visitPage(ROUTES.OVERVIEW, () => {
      cy.get(DATA_IDS.ADD_SOURCE).parent().parent().parent().find(DATA_IDS.CHECKBOX).click();
      cy.get(DATA_IDS.MULTI_SOURCE_CONTROL).should('exist');
      cy.get(DATA_IDS.MULTI_SOURCE_CONTROL).find('button').contains(BUTTONS.UNINSTRUMENT).click();
      cy.get(DATA_IDS.MODAL).should('exist');
      cy.get(DATA_IDS.APPROVE).click();

      // Wait for the uninstrumentation to complete
      cy.wait(3000);
    });
  });

  it('Should cleanup remaining source CRDs', () => {
    cy.exec(`kubectl delete ${sourceCrdName} --all -n ${namespace}`, { failOnNonZeroExit: false });
    cy.exec(`kubectl delete ${configCrdName} --all -n ${namespace}`, { failOnNonZeroExit: false });

    cy.wait(3000).then(() => {
      getCrdIds({ namespace, crdName: sourceCrdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
    });
  });

  it(`Should have 0 ${configCrdName} CRDs in the cluster`, () => {
    getCrdIds({ namespace, crdName: configCrdName, expectedError: TEXTS.NO_RESOURCES(namespace), expectedLength: 0 });
  });
});
