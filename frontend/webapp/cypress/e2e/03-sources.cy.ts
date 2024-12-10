import { BUTTONS, CRD_IDS, CRD_NAMES, DATA_IDS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

describe('Sources CRUD', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it('Should create a CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    cy.exec(`kubectl get ${CRD_NAMES.SOURCE} -n ${NAMESPACES.DEFAULT} | awk 'NR>1 {print $1}'`).then((crdListBefore) => {
      expect(crdListBefore.stderr).to.eq(TEXTS.NO_RESOURCES(NAMESPACES.DEFAULT));
      expect(crdListBefore.stdout).to.eq('');

      const crdIdsBefore = crdListBefore.stdout.split('\n').filter((str) => !!str);
      expect(crdIdsBefore.length).to.eq(0);

      cy.get(DATA_IDS.ADD_ENTITY).click();
      cy.get(DATA_IDS.ADD_SOURCE).click();
      cy.get(DATA_IDS.MODAL_ADD_SOURCE).should('exist');
      cy.get(DATA_IDS.SELECT_NAMESPACE).find(DATA_IDS.CHECKBOX).click();

      // Wait for 3 seconds to allow the namespace & it's resources to be loaded into the UI
      cy.wait(3000).then(() => {
        cy.contains('button', BUTTONS.DONE).click();

        cy.wait('@gql').then(() => {
          cy.exec(`kubectl get ${CRD_NAMES.SOURCE} -n ${NAMESPACES.DEFAULT} | awk 'NR>1 {print $1}'`).then((crdListAfter) => {
            expect(crdListAfter.stderr).to.eq('');
            expect(crdListAfter.stdout).to.not.be.empty;

            const crdIdsAfter = crdListAfter.stdout.split('\n').filter((str) => !!str);
            expect(crdIdsAfter.length).to.eq(5);
          });
        });
      });
    });
  });

  it('Should update the CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    cy.contains(DATA_IDS.SOURCE_NODE, SELECTED_ENTITIES.SOURCE).should('exist').click();
    cy.get(DATA_IDS.DRAWER).should('exist');
    cy.get(DATA_IDS.DRAWER_EDIT).click();
    cy.get(DATA_IDS.SOURCE_TITLE).clear().type(TEXTS.UPDATED_NAME);
    cy.get(DATA_IDS.DRAWER_SAVE).click();
    cy.get(DATA_IDS.DRAWER_CLOSE).click();

    cy.wait('@gql').then(() => {
      cy.exec(`kubectl get ${CRD_NAMES.SOURCE} -n ${NAMESPACES.DEFAULT} | awk 'NR>1 {print $1}'`).then((crdList) => {
        expect(crdList.stderr).to.eq('');
        expect(crdList.stdout).to.not.be.empty;

        const crdIds = crdList.stdout.split('\n').filter((str) => !!str);
        const crdId = CRD_IDS.SOURCE;
        expect(crdIds.length).to.eq(5);
        expect(crdIds).includes(crdId);

        cy.exec(`kubectl get ${CRD_NAMES.SOURCE} ${crdId} -n ${NAMESPACES.DEFAULT} -o json`).then((crd) => {
          expect(crd.stderr).to.eq('');
          expect(crd.stdout).to.not.be.empty;

          const parsed = JSON.parse(crd.stdout);
          const { spec } = parsed?.items?.[0] || parsed || {};

          expect(spec).to.not.be.empty;
          expect(spec.serviceName).to.eq(TEXTS.UPDATED_NAME);
        });
      });
    });
  });

  it('Should delete the CRD from the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    cy.get(DATA_IDS.SOURCE_NODE_HEADER).find(DATA_IDS.CHECKBOX).click();
    cy.get(DATA_IDS.MULTI_SOURCE_CONTROL).should('exist').find('button').contains(BUTTONS.UNINSTRUMENT).click();
    cy.get(DATA_IDS.MODAL).contains(TEXTS.SOURCE_WARN_MODAL_TITLE).should('exist');
    cy.get(DATA_IDS.MODAL).contains(TEXTS.SOURCE_WARN_MODAL_NOTE).should('exist');
    cy.get(DATA_IDS.APPROVE).click();

    cy.wait('@gql').then(() => {
      cy.exec(`kubectl get ${CRD_NAMES.SOURCE} -n ${NAMESPACES.DEFAULT} | awk 'NR>1 {print $1}'`).then((crdList) => {
        expect(crdList.stderr).to.eq(TEXTS.NO_RESOURCES(NAMESPACES.DEFAULT));
        expect(crdList.stdout).to.eq('');

        const crdIds = crdList.stdout.split('\n').filter((str) => !!str);
        expect(crdIds.length).to.eq(0);
      });
    });
  });
});
