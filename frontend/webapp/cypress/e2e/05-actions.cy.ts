import { ROUTES } from '../../utils/constants/routes';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

describe('Actions CRUD', () => {
  const namespace = 'odigos-system';
  const crdName = 'piimaskings.actions.odigos.io';
  const noResourcesFound = `No resources found in ${namespace} namespace.`;

  beforeEach(() => {
    cy.intercept('/graphql').as('gql');
  });

  it('Should create a CRD in the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    cy.exec(`kubectl get ${crdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then((crdListBefore) => {
      expect(crdListBefore.stderr).to.eq(noResourcesFound);
      expect(crdListBefore.stdout).to.eq('');

      const crdIdsBefore = crdListBefore.stdout.split('\n').filter((str) => !!str);
      expect(crdIdsBefore.length).to.eq(0);

      cy.get('[data-id=add-entity]').click();
      cy.get('[data-id=add-action]').click();
      cy.get('[data-id=modal-Add-Action]').should('exist');
      cy.get('[data-id=modal-Add-Action]').find('input').should('have.attr', 'placeholder', 'Type to search...').click();
      cy.get('[data-id=option-pii-masking]').click();
      cy.get('button').contains('DONE').click();

      cy.wait('@gql').then(() => {
        cy.exec(`kubectl get ${crdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then((crdListAfter) => {
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

    const node = cy.contains('[data-id=action-0]', 'PiiMasking');
    expect(node).to.exist;
    node.click();

    cy.get('[data-id=drawer]').should('exist');
    cy.get('button[data-id=drawer-edit]').click();
    cy.get('input[data-id=title]').clear().type('Cypress Test');
    cy.get('button[data-id=drawer-save]').click();
    cy.get('button[data-id=drawer-close]').click();

    cy.wait('@gql').then(() => {
      cy.exec(`kubectl get ${crdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then((crdList) => {
        expect(crdList.stderr).to.eq('');
        expect(crdList.stdout).to.not.be.empty;

        const crdIds = crdList.stdout.split('\n').filter((str) => !!str);
        expect(crdIds.length).to.eq(1);

        cy.exec(`kubectl get ${crdName} ${crdIds[0]} -n ${namespace} -o json`).then((crd) => {
          expect(crd.stderr).to.eq('');
          expect(crd.stdout).to.not.be.empty;

          const parsed = JSON.parse(crd.stdout);
          const { spec } = parsed?.items?.[0] || parsed || {};

          expect(spec).to.not.be.empty;
          expect(spec.actionName).to.eq('Cypress Test');
        });
      });
    });
  });

  it('Should delete the CRD from the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    const node = cy.contains('[data-id=action-0]', 'PiiMasking');
    expect(node).to.exist;
    node.click();

    cy.get('[data-id=drawer]').should('exist');
    cy.get('button[data-id=drawer-edit]').click();
    cy.get('button[data-id=drawer-delete]').click();
    cy.get('[data-id=modal]').contains('Delete action').should('exist');
    cy.get('button[data-id=approve]').click();

    cy.wait('@gql').then(() => {
      cy.exec(`kubectl get ${crdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then((crdList) => {
        expect(crdList.stderr).to.eq(noResourcesFound);
        expect(crdList.stdout).to.eq('');

        const crdIds = crdList.stdout.split('\n').filter((str) => !!str);
        expect(crdIds.length).to.eq(0);
      });
    });
  });
});
