describe('Onboarding', () => {
  it('Visiting the root path fetches a config with GraphQL. A fresh install will result in a redirect to the start of onboarding, confirming Front + Back connections', () => {
    cy.visit('/');
    // If backend connection failed for any reason, teh default redirect would be "/overview"
    cy.location('pathname').should('eq', '/choose-sources');
  });
});
