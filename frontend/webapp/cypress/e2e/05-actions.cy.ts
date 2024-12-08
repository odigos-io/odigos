import { ROUTES } from '../../utils/constants/routes';

describe('Actions CRUD', () => {
  const namespace = 'odigos-system';
  const crdName = 'piimaskings.actions.odigos.io';

  let newCrdId = '';
  let numOfCrds = 0;
  // The number of CRDs that existed in the cluster before running any tests (should be 0).
  // Tests will fail if you have existing CRDs in the cluster.
  // If you have to run tests locally, make sure to clean up the cluster before running the tests.

  it('Should create a CRD in the cluster', () => {
    cy.intercept('/graphql').as('gql');
    cy.visit(ROUTES.OVERVIEW);

    cy.exec(`kubectl get ${crdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then((execBefore) => {
      expect(execBefore.stderr).to.eq(`No resources found in ${namespace} namespace.`);

      const crdIdsBefore = execBefore.stdout.split('\n').filter((str) => !!str);
      numOfCrds = crdIdsBefore.length;
      expect(numOfCrds).to.eq(0);

      cy.get('#add-entity').click();
      cy.get('#add-action').click();

      cy.get('#modal-Add-Action').should('exist');
      cy.get('#modal-Add-Action').find('input').should('have.attr', 'placeholder', 'Type to search...').click();
      cy.get('#option-pii-masking').click();
      cy.get('button').contains('DONE').click();

      cy.wait(3000).then(() => {
        cy.exec(`kubectl get ${crdName} -n ${namespace} | awk 'NR>1 {print $1}'`).then((execAfter) => {
          expect(execAfter.stderr).to.eq('');
          expect(execAfter.stdout).to.not.be.empty;

          const crdIdsAfter = execAfter.stdout.split('\n').filter((str) => !!str);
          numOfCrds = crdIdsAfter.length;
          expect(numOfCrds).to.eq(1);
          newCrdId = crdIdsAfter.filter((id) => !crdIdsBefore.includes(id))[0];
          expect(newCrdId).to.not.be.empty;
        });
      });
    });
  });

  it('Should update the CRD in the cluster', () => {
    cy.intercept('/graphql').as('gql');
    cy.visit(ROUTES.OVERVIEW);

    const node = cy.contains('[data-id=action-0]', 'PiiMasking');
    expect(node).to.exist;
    node.click();

    cy.get('#drawer').should('exist');
    cy.get('button#drawer-edit').click();
    cy.get('input#title').clear().type('Cypress Test');
    cy.get('button#drawer-save').click();
    cy.get('button#drawer-close').click();

    cy.wait(3000).then(() => {
      cy.exec(`kubectl get ${crdName} ${newCrdId} -n ${namespace} -o json`).then(({ stderr, stdout }) => {
        expect(stderr).to.eq('');
        expect(stdout).to.not.be.empty;

        const { spec } = JSON.parse(stdout)?.items?.[0] || {};
        expect(spec).to.exist;
        expect(spec.actionName).to.eq('Cypress Test');
      });
    });
  });
});
