import { ROUTES } from '../../utils/constants/routes';

describe('Onboarding', () => {
  it('Should contain at least a "default" namespace', () => {
    cy.intercept('/graphql').as('gql');
    cy.visit(ROUTES.CHOOSE_SOURCES);

    cy.wait('@gql').then(() => {
      expect('#namespace-default').to.exist;
    });
  });

  it('Should contain at least a "Jaeger" destination', () => {
    cy.intercept('/graphql').as('gql');
    cy.visit(ROUTES.CHOOSE_DESTINATION);

    cy.wait('@gql').then(() => {
      cy.get('button').contains('ADD DESTINATION').click();
      expect('#destination-jaeger').to.exist;
    });
  });

  it('Should autocomplete the "Jaeger" destination', () => {
    cy.intercept('/graphql').as('gql');
    cy.visit(ROUTES.CHOOSE_DESTINATION);

    cy.wait('@gql').then(() => {
      cy.get('button').contains('ADD DESTINATION').click();
      cy.get('#destination-jaeger').click();
      expect('#JAEGER_URL').to.not.be.empty;
    });
  });

  it('Should allow the user to pass every step, and end-up on the "Overview" page.', () => {
    cy.visit(ROUTES.CHOOSE_SOURCES);

    cy.get('button').contains('NEXT').click();
    cy.location('pathname').should('eq', ROUTES.CHOOSE_DESTINATION);

    cy.get('button').contains('DONE').click();
    cy.location('pathname').should('eq', ROUTES.OVERVIEW);
  });
});
