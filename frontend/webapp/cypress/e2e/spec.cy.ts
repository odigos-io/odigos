describe('Onboarding', () => {
  it('Visiting the root path with a fresh install will result in a redirect to the start of onboarding', () => {
    cy.visit('/');
    cy.location('pathname').should('eq', '/choose-sources');
  });
});
