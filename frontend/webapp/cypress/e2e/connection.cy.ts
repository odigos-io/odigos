describe('Root Connection', () => {
  it('Should fetch a config with GraphQL. A redirect of any kind confirms Frontend + Backend connections.', () => {
    cy.intercept('/graphql').as('gql');
    cy.visit('/');

    cy.wait('@gql').then(() => {
      cy.location().should((loc) => {
        // If GraphQL failed to fetch the config, the app will remain on "/", thereby failing the test.
        expect(loc.pathname).to.be.oneOf(['/choose-sources', '/overview']);
      });
    });
  });
});
