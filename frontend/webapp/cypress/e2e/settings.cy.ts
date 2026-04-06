import { CONFIG_MAPS, DATA_IDS, NAMESPACES, ROUTES, TEXTS } from '../constants';
import { awaitToast, handleExceptions, visitPage } from '../functions';

const namespace = NAMESPACES.ODIGOS;
const testClusterName = 'cypress-e2e-test';
let originalClusterName = '';

const getConfigMapYaml = (configMapName: string, callback: (yaml: string) => void) => {
  cy.exec(`kubectl get configmap ${configMapName} -n ${namespace} -o jsonpath='{.data.config\\.yaml}'`).then(({ stdout }) => {
    callback(stdout);
  });
};

const clickToggle = (fieldPath: string) => {
  // The data-id is on the Container div, but the onClick is on the Controller (first child).
  // Clicking Container center can land on the label text and miss the switch.
  cy.get(DATA_IDS.SETTINGS_FIELD(fieldPath)).children().first().click();
};

const setInput = (fieldPath: string, value: string) => {
  cy.get(DATA_IDS.SETTINGS_FIELD(fieldPath)).click().focused().clear().type(value);
};

const selectDropdownOption = (fieldPath: string, optionLabel: string) => {
  cy.get(DATA_IDS.SETTINGS_FIELD(fieldPath)).click();
  // Navigate from <input> → InputWrapper → FlexColumn → Relative (DropData root)
  // to scope the search to only this dropdown's popup options.
  cy.get(DATA_IDS.SETTINGS_FIELD(fieldPath)).parent().parent().parent().contains(optionLabel).click();
};

const addMultiInputValue = (fieldPath: string, value: string) => {
  cy.get(DATA_IDS.SETTINGS_FIELD(fieldPath)).find('button').first().click();
  cy.get(DATA_IDS.SETTINGS_FIELD(fieldPath)).find('input').last().clear().type(value);
};

describe('Settings CRUD', () => {
  beforeEach(() => {
    cy.intercept('/graphql').as('gql');
    handleExceptions();
  });

  // ── Setup ─────────────────────────────────────────────────────────────────

  it('Should capture initial state from the cluster', () => {
    // Reset any leftover test state from previous runs before capturing the baseline
    cy.exec(`kubectl delete configmap ${CONFIG_MAPS.LOCAL_UI_CONFIG} -n ${namespace}`, { failOnNonZeroExit: false });
    cy.wait(10000);

    getConfigMapYaml(CONFIG_MAPS.EFFECTIVE_CONFIG, (yaml) => {
      const match = yaml.match(/clusterName:\s*(.+)/);
      originalClusterName = match ? match[1].trim() : '';
    });
  });

  // ── Read ──────────────────────────────────────────────────────────────────

  it('Should render config sections from the cluster', () => {
    visitPage(ROUTES.SETTINGS, () => {
      cy.wait('@gql').then(() => {
        cy.contains('General').should('exist');
        cy.contains('Instrumentation').should('exist');
        cy.contains('Rollout & Rollback').should('exist');
        cy.contains('Namespaces & Filtering').should('exist');
        cy.contains('Component Log Levels').should('exist');
        cy.contains('Sampling').should('exist');
        cy.contains('Advanced').should('exist');
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
        const cancelTestValue = (originalClusterName || 'cluster') + '-cancel-test';

        cy.get(DATA_IDS.SETTINGS_FIELD('clusterName')).click().focused().clear().type(cancelTestValue);

        cy.get(DATA_IDS.SETTINGS_SAVE).should('be.visible');
        cy.get(DATA_IDS.SETTINGS_CANCEL).should('be.visible');

        cy.get(DATA_IDS.SETTINGS_CANCEL).click();

        cy.get(DATA_IDS.SETTINGS_FIELD('clusterName')).should('have.value', originalClusterName);
        cy.get(DATA_IDS.SETTINGS_SAVE).should('not.exist');
      });
    });
  });

  // ── Update ALL non-helm-only fields + Save ────────────────────────────────

  it('Should update all non-helm-only fields and save successfully', () => {
    visitPage(ROUTES.SETTINGS, () => {
      cy.wait('@gql').then(() => {
        // ─ General ─
        setInput('clusterName', testClusterName);
        clickToggle('telemetryEnabled');

        // ─ Instrumentation ─
        // Note: instrumentor.agentEnvVarsInjectionMethod dropdown is skipped because
        // the config YAML options use hyphens while the GraphQL enum uses underscores.
        clickToggle('allowConcurrentAgents.enabled');
        clickToggle('instrumentor.checkDeviceHealthBeforeInjection');
        clickToggle('wasp.enabled');

        // ─ Rollout & Rollback ─
        clickToggle('rollout.automaticRolloutDisabled');
        setInput('rollout.maxConcurrentRollouts', '5');
        clickToggle('autoRollback.disabled');
        setInput('autoRollback.graceTime', '60s');
        setInput('autoRollback.stabilityWindowTime', '120s');

        // ─ Namespaces & Filtering ─
        addMultiInputValue('ignoredNamespaces', 'cypress-test-ns');
        addMultiInputValue('ignoredContainers', 'cypress-test-container');
        clickToggle('ignoreOdigosNamespace');

        // ─ Component Log Levels ─
        selectDropdownOption('componentLogLevels.default', 'info');
        selectDropdownOption('componentLogLevels.autoscaler', 'info');
        selectDropdownOption('componentLogLevels.scheduler', 'info');
        selectDropdownOption('componentLogLevels.instrumentor', 'info');
        selectDropdownOption('componentLogLevels.odiglet', 'info');
        selectDropdownOption('componentLogLevels.deviceplugin', 'info');
        selectDropdownOption('componentLogLevels.ui', 'info');
        selectDropdownOption('componentLogLevels.collector', 'info');

        // ─ Sampling ─
        clickToggle('sampling.dryRun');
        clickToggle('sampling.spanSamplingAttributes.disabled');
        clickToggle('sampling.spanSamplingAttributes.samplingCategoryDisabled');
        clickToggle('sampling.spanSamplingAttributes.traceDecidingRuleDisabled');
        clickToggle('sampling.spanSamplingAttributes.spanDecisionAttributesDisabled');
        clickToggle('sampling.tailSampling.disabled');
        setInput('sampling.tailSampling.traceAggregationWaitDuration', '45s');
        clickToggle('sampling.k8sHealthProbesSampling.enabled');
        setInput('sampling.k8sHealthProbesSampling.keepPercentage', '50');

        // ─ Advanced ─
        setInput('goAutoOffsetsCron', '0 0 * * *');
        setInput('goAutoOffsetsMode', 'cypress-test');

        // ─ Save ─
        cy.get(DATA_IDS.SETTINGS_SAVE).should('be.visible').click();

        // Wait for the mutation to complete and toast to appear.
        // Don't use cy.wait('@gql') here — background SSE-triggered queries
        // can consume the alias before the mutation response arrives.
        awaitToast({ message: TEXTS.NOTIF_CONFIG_UPDATED });
      });
    });
  });

  // ── Verify kubectl ────────────────────────────────────────────────────────

  it(`Should have written the changes to the ${CONFIG_MAPS.LOCAL_UI_CONFIG} ConfigMap`, () => {
    getConfigMapYaml(CONFIG_MAPS.LOCAL_UI_CONFIG, (yaml) => {
      // ─ General (input) ─
      expect(yaml).to.contain(`clusterName: ${testClusterName}`);

      // ─ Instrumentation (toggles — key presence confirms the toggle fired) ─
      expect(yaml).to.contain('allowConcurrentAgents:');
      expect(yaml).to.contain('checkDeviceHealthBeforeInjection:');
      expect(yaml).to.contain('waspEnabled:');

      // ─ Rollout & Rollback (inputs + toggles) ─
      expect(yaml).to.contain('automaticRolloutDisabled:');
      expect(yaml).to.contain('maxConcurrentRollouts: 5');
      expect(yaml).to.contain('rollbackDisabled:');
      expect(yaml).to.satisfy((s: string) => s.includes('rollbackGraceTime: 60s') || s.includes('rollbackGraceTime: "60s"'));
      expect(yaml).to.satisfy((s: string) => s.includes('rollbackStabilityWindow: 120s') || s.includes('rollbackStabilityWindow: "120s"'));

      // ─ Namespaces & Filtering (multiInputs + toggle) ─
      expect(yaml).to.contain('cypress-test-ns');
      expect(yaml).to.contain('cypress-test-container');
      expect(yaml).to.contain('ignoreOdigosNamespace:');

      // ─ Advanced (inputs) ─
      expect(yaml).to.contain('goAutoOffsetsCron: 0 0 * * *');
      expect(yaml).to.contain('goAutoOffsetsMode: cypress-test');
    });
  });

  it(`Should have reconciled changes into the ${CONFIG_MAPS.EFFECTIVE_CONFIG} ConfigMap`, () => {
    cy.wait(5000).then(() => {
      getConfigMapYaml(CONFIG_MAPS.EFFECTIVE_CONFIG, (yaml) => {
        // ─ General (input) ─
        expect(yaml).to.contain(`clusterName: ${testClusterName}`);

        // ─ Instrumentation (toggles) ─
        expect(yaml).to.contain('allowConcurrentAgents:');
        expect(yaml).to.contain('checkDeviceHealthBeforeInjection:');
        expect(yaml).to.contain('waspEnabled:');

        // ─ Rollout & Rollback (inputs + toggles) ─
        expect(yaml).to.contain('automaticRolloutDisabled:');
        expect(yaml).to.contain('maxConcurrentRollouts: 5');
        expect(yaml).to.contain('rollbackDisabled:');
        expect(yaml).to.satisfy((s: string) => s.includes('rollbackGraceTime: 60s') || s.includes('rollbackGraceTime: "60s"'));
        expect(yaml).to.satisfy((s: string) => s.includes('rollbackStabilityWindow: 120s') || s.includes('rollbackStabilityWindow: "120s"'));

        // ─ Namespaces & Filtering (multiInputs) ─
        expect(yaml).to.contain('cypress-test-ns');
        expect(yaml).to.contain('cypress-test-container');

        // ─ Advanced (inputs) ─
        expect(yaml).to.contain('goAutoOffsetsCron: 0 0 * * *');
        expect(yaml).to.contain('goAutoOffsetsMode: cypress-test');
      });
    });
  });

  // ── Cleanup ───────────────────────────────────────────────────────────────

  it('Should restore the original ConfigMap and reconcile', () => {
    cy.exec(`kubectl delete configmap ${CONFIG_MAPS.LOCAL_UI_CONFIG} -n ${namespace}`, { failOnNonZeroExit: false });

    cy.wait(10000).then(() => {
      getConfigMapYaml(CONFIG_MAPS.EFFECTIVE_CONFIG, (yaml) => {
        expect(yaml).to.not.contain(`clusterName: ${testClusterName}`);
        expect(yaml).to.not.contain('goAutoOffsetsMode: cypress-test');
        expect(yaml).to.not.contain('cypress-test-ns');
        expect(yaml).to.not.contain('cypress-test-container');
        expect(yaml).to.not.contain('maxConcurrentRollouts: 5');
      });
    });
  });
});
