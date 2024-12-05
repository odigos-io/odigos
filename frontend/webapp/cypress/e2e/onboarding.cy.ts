describe('Onboarding', () => {
  it('Should contain at least a "default" namespace', () => {
    cy.visit('/choose-sources');

    cy.get('#no-data').should('not.exist');
    cy.get('#namespace-default').should('exist');
  });

  it('Should contain at least a "Jaeger" destination', () => {
    cy.visit('/choose-destination');

    cy.get('button').contains('ADD DESTINATION').click();
    cy.get('#no-data').should('not.exist');

    cy.get('input').should('have.attr', 'placeholder', 'Search...').type('Jaeger');
    cy.get('#destination-jaeger').should('exist');
  });

  it('Should allow the user to pass every step, and end-up on the Overview page.', () => {
    cy.visit('/choose-sources');

    cy.get('button').contains('NEXT').click();
    cy.location('pathname').should('eq', '/choose-destination');

    cy.get('button').contains('DONE').click();
    cy.location('pathname').should('eq', '/overview');
  });
});
