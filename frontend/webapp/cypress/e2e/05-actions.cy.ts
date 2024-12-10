import { BUTTONS, CRD_NAMES, DATA_IDS, INPUTS, NAMESPACES, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

describe('Actions CRUD', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it('Should create a CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    cy.exec(`kubectl get ${CRD_NAMES.ACTION} -n ${NAMESPACES.ODIGOS_SYSTEM} | awk 'NR>1 {print $1}'`).then((crdListBefore) => {
      expect(crdListBefore.stderr).to.eq(TEXTS.NO_RESOURCES(NAMESPACES.ODIGOS_SYSTEM));
      expect(crdListBefore.stdout).to.eq('');

      const crdIdsBefore = crdListBefore.stdout.split('\n').filter((str) => !!str);
      expect(crdIdsBefore.length).to.eq(0);

      cy.get(DATA_IDS.ADD_ENTITY).click();
      cy.get(DATA_IDS.ADD_ACTION).click();
      cy.get(DATA_IDS.MODAL_ADD_ACTION).should('exist').find('input').should('have.attr', 'placeholder', INPUTS.ACTION_DROPDOWN).click();
      cy.get(DATA_IDS.ACTION_DROPDOWN_OPTION).click();
      cy.get('button').contains(BUTTONS.DONE).click();

      cy.wait('@gql').then(() => {
        cy.exec(`kubectl get ${CRD_NAMES.ACTION} -n ${NAMESPACES.ODIGOS_SYSTEM} | awk 'NR>1 {print $1}'`).then((crdListAfter) => {
          expect(crdListAfter.stderr).to.eq('');
          expect(crdListAfter.stdout).to.not.be.empty;

          const crdIdsAfter = crdListAfter.stdout.split('\n').filter((str) => !!str);
          expect(crdIdsAfter.length).to.eq(1);
        });
      });
    });
  });

  it('Should update the CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    cy.contains(DATA_IDS.ACTION_NODE, SELECTED_ENTITIES.ACTION).should('exist').click();
    cy.get(DATA_IDS.DRAWER).should('exist');
    cy.get(DATA_IDS.DRAWER_EDIT).click();
    cy.get(DATA_IDS.TITLE).clear().type(TEXTS.UPDATED_NAME);
    cy.get(DATA_IDS.DRAWER_SAVE).click();
    cy.get(DATA_IDS.DRAWER_CLOSE).click();

    cy.wait('@gql').then(() => {
      cy.exec(`kubectl get ${CRD_NAMES.ACTION} -n ${NAMESPACES.ODIGOS_SYSTEM} | awk 'NR>1 {print $1}'`).then((crdList) => {
        expect(crdList.stderr).to.eq('');
        expect(crdList.stdout).to.not.be.empty;

        const crdIds = crdList.stdout.split('\n').filter((str) => !!str);
        const crdId = crdIds[0];
        expect(crdIds.length).to.eq(1);
        expect(crdIds).includes(crdId);

        cy.exec(`kubectl get ${CRD_NAMES.ACTION} ${crdId} -n ${NAMESPACES.ODIGOS_SYSTEM} -o json`).then((crd) => {
          expect(crd.stderr).to.eq('');
          expect(crd.stdout).to.not.be.empty;

          const parsed = JSON.parse(crd.stdout);
          const { spec } = parsed?.items?.[0] || parsed || {};

          expect(spec).to.not.be.empty;
          expect(spec.actionName).to.eq(TEXTS.UPDATED_NAME);
        });
      });
    });
  });

  it('Should delete the CRD from the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    cy.contains(DATA_IDS.ACTION_NODE, SELECTED_ENTITIES.ACTION).should('exist').click();
    cy.get(DATA_IDS.DRAWER).should('exist');
    cy.get(DATA_IDS.DRAWER_EDIT).click();
    cy.get(DATA_IDS.DRAWER_DELETE).click();
    cy.get(DATA_IDS.MODAL).contains(TEXTS.ACTION_WARN_MODAL_TITLE).should('exist');
    cy.get(DATA_IDS.APPROVE).click();

    cy.wait('@gql').then(() => {
      cy.exec(`kubectl get ${CRD_NAMES.ACTION} -n ${NAMESPACES.ODIGOS_SYSTEM} | awk 'NR>1 {print $1}'`).then((crdList) => {
        expect(crdList.stderr).to.eq(TEXTS.NO_RESOURCES(NAMESPACES.ODIGOS_SYSTEM));
        expect(crdList.stdout).to.eq('');

        const crdIds = crdList.stdout.split('\n').filter((str) => !!str);
        expect(crdIds.length).to.eq(0);
      });
    });
  });
});
