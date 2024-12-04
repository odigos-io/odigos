describe('Onboarding', () => {
  it('Visiting the root path fetches a config with GraphQL. A fresh install will result in a redirect to the start of onboarding, confirming Front + Back connections', () => {
    cy.visit('/');
    cy.location('pathname').should('eq', '/choose-sources');
  });
});
