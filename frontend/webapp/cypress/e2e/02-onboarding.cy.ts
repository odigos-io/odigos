import { CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';
import { getCrdIds, handleExceptions, visitPage, waitForGraphqlOperation } from '../functions';

describe('Onboarding', () => {
  beforeEach(() => {
    cy.intercept('/graphql').as('gql');
    handleExceptions();
  });

  // ── Skip onboarding ───────────────────────────────────────────────────────

  it('Should skip onboarding without setting up anything and reach the overview', () => {
    visitPage(ROUTES.ONBOARDING, () => {
      cy.get(DATA_IDS.ONBOARDING_GET_STARTED).click();

      // Step 2: Sources
      cy.contains('Add Source').should('be.visible');
      cy.get(DATA_IDS.WIDE_DRAWER_SKIP).should('be.visible').click();

      // Step 3: Destinations
      cy.contains('Add Destinations').should('be.visible');
      cy.get(DATA_IDS.WIDE_DRAWER_SKIP).should('be.visible').click();

      // Step 4: Summary
      cy.contains('Summary').should('be.visible');
      cy.get(DATA_IDS.WIDE_DRAWER_SAVE).should('be.visible').click();

      cy.location('pathname').should('eq', ROUTES.OVERVIEW);
    });
  });

  it('Should have 0 source CRDs after skipping onboarding', () => {
    getCrdIds({ namespace: NAMESPACES.APPS, crdName: CRD_NAMES.SOURCE, expectedError: TEXTS.NO_RESOURCES(NAMESPACES.APPS), expectedLength: 0 });
  });

  it('Should have 0 destination CRDs after skipping onboarding', () => {
    getCrdIds({ namespace: NAMESPACES.ODIGOS, crdName: CRD_NAMES.DESTINATION, expectedError: TEXTS.NO_RESOURCES(NAMESPACES.ODIGOS), expectedLength: 0 });
  });

  // ── Full onboarding with sources + destination ────────────────────────────

  it('Should complete onboarding with sources and a destination selected', () => {
    visitPage(ROUTES.ONBOARDING, () => {
      cy.get(DATA_IDS.ONBOARDING_GET_STARTED).click();

      // Step 2: Sources — select the "default" namespace
      cy.contains('Add Source').should('be.visible');
      waitForGraphqlOperation('GetNamespacesWithWorkloads').then(() => {
        cy.get(DATA_IDS.SELECT_NAMESPACE).should('be.visible').click();

        // Click the namespace checkbox to select all its workloads
        cy.get(DATA_IDS.SELECT_NAMESPACE).find(DATA_IDS.CHECKBOX).click();

        // Verify all workloads appear in the right column
        SELECTED_ENTITIES.NAMESPACE_SOURCES.forEach(({ name }) => {
          cy.get(DATA_IDS.SELECT_SOURCE(name)).should('exist');
        });

        cy.get(DATA_IDS.WIDE_DRAWER_NEXT).should('be.visible').click();

        // Step 3: Destinations — add Jaeger
        cy.contains('Add Destinations').should('be.visible');
        waitForGraphqlOperation('GetPotentialDestinations').then(() => {
          cy.contains('Detected by system').should('be.visible');
          cy.get(DATA_IDS.SELECT_DESTINATION).first().click({ force: true });

          // The auto-fill field should be populated from detected destinations
          cy.get(DATA_IDS.SELECT_DESTINATION_AUTOFILL_FIELD).should('have.value', SELECTED_ENTITIES.DESTINATION.AUTOFILL_VALUE);

          cy.get(DATA_IDS.DEST_FORM_ADD).click();
          cy.get(DATA_IDS.WIDE_DRAWER_NEXT).should('be.visible').click();

          // Step 4: Summary
          cy.contains('Summary').should('be.visible');
          cy.get(DATA_IDS.WIDE_DRAWER_SAVE).should('be.visible').click();

          cy.location('pathname').should('eq', ROUTES.OVERVIEW);
        });
      });
    });
  });

  // ── Verify CRDs ──────────────────────────────────────────────────────────

  it(`Should have ${CRD_NAMES.SOURCE} CRDs in the cluster`, () => {
    // Namespace-level auto-instrument creates 1 source CRD for the namespace
    getCrdIds({ namespace: NAMESPACES.APPS, crdName: CRD_NAMES.SOURCE, expectedError: '', expectedLength: 1 });
  });

  it(`Should have ${CRD_NAMES.INSTRUMENTATION_CONFIG} CRDs in the cluster`, () => {
    // Each workload in the namespace gets its own instrumentationconfig
    cy.exec(`kubectl get ${CRD_NAMES.INSTRUMENTATION_CONFIG} -n ${NAMESPACES.APPS} | awk 'NR>1 {print $1}'`).then(({ stdout }) => {
      const count = stdout.split('\n').filter((s) => !!s).length;
      expect(count).to.be.gte(SELECTED_ENTITIES.NAMESPACE_SOURCES.length);
    });
  });

  it(`Should have 1 ${CRD_NAMES.DESTINATION} CRD in the cluster`, () => {
    getCrdIds({ namespace: NAMESPACES.ODIGOS, crdName: CRD_NAMES.DESTINATION, expectedError: '', expectedLength: 1 });
  });

  // ── Cleanup ───────────────────────────────────────────────────────────────

  it('Should cleanup sources created during onboarding', () => {
    cy.exec(`kubectl delete ${CRD_NAMES.SOURCE} --all -n ${NAMESPACES.APPS}`, { failOnNonZeroExit: false });
    cy.exec(`kubectl delete ${CRD_NAMES.INSTRUMENTATION_CONFIG} --all -n ${NAMESPACES.APPS}`, { failOnNonZeroExit: false });

    cy.wait(3000).then(() => {
      getCrdIds({ namespace: NAMESPACES.APPS, crdName: CRD_NAMES.SOURCE, expectedError: TEXTS.NO_RESOURCES(NAMESPACES.APPS), expectedLength: 0 });
    });
  });

  it('Should cleanup destinations created during onboarding', () => {
    cy.exec(`kubectl delete ${CRD_NAMES.DESTINATION} --all -n ${NAMESPACES.ODIGOS}`, { failOnNonZeroExit: false });

    cy.wait(3000).then(() => {
      getCrdIds({ namespace: NAMESPACES.ODIGOS, crdName: CRD_NAMES.DESTINATION, expectedError: TEXTS.NO_RESOURCES(NAMESPACES.ODIGOS), expectedLength: 0 });
    });
  });
});
