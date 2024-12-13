import { ROUTES } from '../constants';

describe('Root Connection', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it('Should fetch a config with GraphQL. A redirect of any kind confirms Frontend + Backend connections.', () => {
    cy.visit(ROUTES.ROOT);
    cy.wait('@gql').then(() => {
      cy.location().should((loc) => {
        // If GraphQL failed to fetch the config, the app will remain on "/", thereby failing the test.
        expect(loc.pathname).to.be.oneOf([ROUTES.CHOOSE_SOURCES, ROUTES.OVERVIEW]);
      });
    });
  });
});
