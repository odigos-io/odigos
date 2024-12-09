import { ROUTES } from '../../utils/constants/routes';

// The number of CRDs that exist in the cluster before running any tests should be 0.
// Tests will fail if you have existing CRDs in the cluster.
// If you have to run tests locally, make sure to clean up the cluster before running the tests.

describe('Sources CRUD', () => {
  const namespace = 'default';
  const crdName = 'instrumentationconfigs.odigos.io';
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
      cy.get('[data-id=add-source]').click();
      cy.get('[data-id=modal-Add-Source]').should('exist');
      cy.get('[data-id=namespace-default]').find('[data-id=checkbox]').click();

      // Wait for 3 seconds to allow the namespace & it's resources to be loaded into the UI
      cy.wait(3000).then(() => {
        cy.get('button').contains('DONE').click();

        cy.wait('@gql').then(() => {
          cy.exec(`kubectl get ${crdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then((crdListAfter) => {
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

    const node = cy.contains('[data-id=source-1]', 'frontend');
    expect(node).to.exist;
    node.click();

    cy.get('[data-id=drawer]').should('exist');
    cy.get('button[data-id=drawer-edit]').click();
    cy.get('input[data-id=sourceName]').clear().type('Cypress Test');
    cy.get('button[data-id=drawer-save]').click();
    cy.get('button[data-id=drawer-close]').click();

    cy.wait('@gql').then(() => {
      cy.exec(`kubectl get ${crdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then((crdList) => {
        expect(crdList.stderr).to.eq('');
        expect(crdList.stdout).to.not.be.empty;

        const crdIds = crdList.stdout.split('\n').filter((str) => !!str);
        const crdId = 'deployment-frontend';
        expect(crdIds.length).to.eq(5);
        expect(crdIds).includes(crdId);

        cy.exec(`kubectl get ${crdName} ${crdId} -n ${namespace} -o json`).then((crd) => {
          expect(crd.stderr).to.eq('');
          expect(crd.stdout).to.not.be.empty;

          const parsed = JSON.parse(crd.stdout);
          const { spec } = parsed?.items?.[0] || parsed || {};

          expect(spec).to.not.be.empty;
          expect(spec.serviceName).to.eq('Cypress Test');
        });
      });
    });
  });

  it('Should delete the CRD from the cluster', () => {
    cy.visit(ROUTES.OVERVIEW);

    cy.get('[data-id=source-header]').find('[data-id=checkbox]').click();
    cy.get('[data-id=multi-source-control]').should('exist');
    cy.get('[data-id=multi-source-control]').find('button').contains('Uninstrument').click();
    cy.get('[data-id=modal]').contains('Uninstrument 5 sources').should('exist');
    cy.get('[data-id=modal]').contains("You're about to uninstrument the last source").should('exist');
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
