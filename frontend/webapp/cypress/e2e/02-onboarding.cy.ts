import { ROUTES } from '../../utils/constants/routes';

describe('Onboarding', () => {
  beforeEach(() => {
    cy.intercept('/graphql').as('gql');
  });

  it('Should contain at least a "default" namespace', () => {
    cy.visit(ROUTES.CHOOSE_SOURCES);
    cy.wait('@gql').then(() => {
      cy.get('[data-id=namespace-default]').contains('default').should('exist');
    });
  });

  it('Should contain at least a "Jaeger" destination', () => {
    cy.visit(ROUTES.CHOOSE_DESTINATION);
    cy.contains('button', 'ADD DESTINATION').click();
    cy.wait('@gql').then(() => {
      cy.get('[data-id=destination-jaeger]').contains('Jaeger').should('exist');
    });
  });

  it('Should autocomplete the "Jaeger" destination', () => {
    cy.visit(ROUTES.CHOOSE_DESTINATION);
    cy.contains('button', 'ADD DESTINATION').click();
    cy.wait('@gql').then(() => {
      cy.get('[data-id=destination-jaeger]').contains('Jaeger').click();
      expect('[data-id=JAEGER_URL]').to.not.be.empty;
    });
  });

  it('Should allow the user to pass every step, and end-up on the "Overview" page.', () => {
    cy.visit(ROUTES.CHOOSE_SOURCES);
    cy.contains('button', 'NEXT').click();
    cy.location('pathname').should('eq', ROUTES.CHOOSE_DESTINATION);
    cy.contains('button', 'DONE').click();
    cy.location('pathname').should('eq', ROUTES.OVERVIEW);
  });
});
