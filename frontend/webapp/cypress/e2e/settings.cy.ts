import { CONFIG_MAPS, DATA_IDS, NAMESPACES, ROUTES, TEXTS } from '../constants';
import { awaitToast, handleExceptions, visitPage } from '../functions';

const namespace = NAMESPACES.ODIGOS_SYSTEM;
const testClusterName = 'cypress-e2e-test';
let originalClusterName = '';

const getConfigMapYaml = (configMapName: string, callback: (yaml: string) => void) => {
  cy.exec(`kubectl get configmap ${configMapName} -n ${namespace} -o jsonpath='{.data.config\\.yaml}'`).then(({ stdout }) => {
    callback(stdout);
  });
};

describe('Settings CRUD', () => {
  beforeEach(() => {
    cy.intercept('/graphql').as('gql');
    handleExceptions();
  });

  // ── Read ──────────────────────────────────────────────────────────────────

  it('Should capture the initial clusterName from the effective-config ConfigMap', () => {
    getConfigMapYaml(CONFIG_MAPS.EFFECTIVE_CONFIG, (yaml) => {
      const match = yaml.match(/clusterName:\s*(.+)/);
      originalClusterName = match ? match[1].trim() : '';
    });
  });

  it('Should render config sections from the cluster', () => {
    visitPage(ROUTES.SETTINGS, () => {
      cy.wait('@gql').then(() => {
        cy.contains('General').should('exist');
        cy.contains('Effective Config YAML').should('exist');
      });
    });
  });

  it('Should render the clusterName field with its current value', () => {
    visitPage(ROUTES.SETTINGS, () => {
      cy.wait('@gql').then(() => {
        cy.get(DATA_IDS.SETTINGS_FIELD('clusterName')).should('exist').and('have.value', originalClusterName);
      });
    });
  });

  it('Should not show the Save/Cancel island when no changes are made', () => {
    visitPage(ROUTES.SETTINGS, () => {
      cy.wait('@gql').then(() => {
        cy.get(DATA_IDS.SETTINGS_SAVE).should('not.exist');
        cy.get(DATA_IDS.SETTINGS_CANCEL).should('not.exist');
      });
    });
  });

  // ── Update + Cancel ───────────────────────────────────────────────────────

  it('Should show the Save/Cancel island when a field is modified, and revert on Cancel', () => {
    visitPage(ROUTES.SETTINGS, () => {
      cy.wait('@gql').then(() => {
        cy.get(DATA_IDS.SETTINGS_FIELD('clusterName')).click().focused().clear().type(testClusterName);

        cy.get(DATA_IDS.SETTINGS_SAVE).should('be.visible');
        cy.get(DATA_IDS.SETTINGS_CANCEL).should('be.visible');

        cy.get(DATA_IDS.SETTINGS_CANCEL).click();

        cy.get(DATA_IDS.SETTINGS_FIELD('clusterName')).should('have.value', originalClusterName);
        cy.get(DATA_IDS.SETTINGS_SAVE).should('not.exist');
      });
    });
  });

  // ── Update + Save ─────────────────────────────────────────────────────────

  it('Should update clusterName via the UI and show a success toast', () => {
    visitPage(ROUTES.SETTINGS, () => {
      cy.wait('@gql').then(() => {
        cy.get(DATA_IDS.SETTINGS_FIELD('clusterName')).click().focused().clear().type(testClusterName);
        cy.get(DATA_IDS.SETTINGS_SAVE).click();

        cy.wait('@gql').then(() => {
          awaitToast({ message: TEXTS.NOTIF_CONFIG_UPDATED });
        });
      });
    });
  });

  it(`Should have written clusterName to the ${CONFIG_MAPS.LOCAL_UI_CONFIG} ConfigMap`, () => {
    getConfigMapYaml(CONFIG_MAPS.LOCAL_UI_CONFIG, (yaml) => {
      expect(yaml).to.contain(`clusterName: ${testClusterName}`);
    });
  });

  it(`Should have reconciled clusterName into the ${CONFIG_MAPS.EFFECTIVE_CONFIG} ConfigMap`, () => {
    cy.wait(5000).then(() => {
      getConfigMapYaml(CONFIG_MAPS.EFFECTIVE_CONFIG, (yaml) => {
        expect(yaml).to.contain(`clusterName: ${testClusterName}`);
      });
    });
  });

  // ── Cleanup ───────────────────────────────────────────────────────────────

  it('Should revert clusterName back to the original value via the UI', () => {
    visitPage(ROUTES.SETTINGS, () => {
      cy.wait('@gql').then(() => {
        const field = cy.get(DATA_IDS.SETTINGS_FIELD('clusterName'));
        field.click().focused().clear();

        if (originalClusterName) {
          field.type(originalClusterName);
        }

        cy.get(DATA_IDS.SETTINGS_SAVE).click();

        cy.wait('@gql').then(() => {
          awaitToast({ message: TEXTS.NOTIF_CONFIG_UPDATED });
        });
      });
    });
  });

  it(`Should have reverted clusterName in the ${CONFIG_MAPS.EFFECTIVE_CONFIG} ConfigMap`, () => {
    cy.wait(5000).then(() => {
      getConfigMapYaml(CONFIG_MAPS.EFFECTIVE_CONFIG, (yaml) => {
        expect(yaml).to.not.contain(`clusterName: ${testClusterName}`);

        if (originalClusterName) {
          expect(yaml).to.contain(`clusterName: ${originalClusterName}`);
        }
      });
    });
  });
});
