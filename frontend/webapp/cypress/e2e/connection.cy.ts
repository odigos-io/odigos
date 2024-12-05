describe('Root Connection', () => {
  it('Should fetch a config with GraphQL. A redirect of any kind confirms Frontend + Backend connections.', () => {
    cy.visit('/');

    // If GraphQL failed to fetch the config, the app will remain on "/", thereby failing the test.
    cy.location().should((loc) => {
      expect(loc.pathname).to.be.oneOf(['/choose-sources', '/overview']);
    });
  });
});
