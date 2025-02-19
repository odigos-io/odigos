import { ROUTES } from '../constants';
import { visitPage } from '../functions';

describe('Root Connection', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it('Should fetch a config with GraphQL, confirming Frontend + Backend connections.', () => {
    visitPage(ROUTES.ROOT, () => {
      // Wait for the config to load
      cy.wait('@gql').then(() => {
        // If GraphQL failed to fetch the config, the app will remain on "/", thereby failing the test.
        cy.location('pathname').should('be.oneOf', [ROUTES.CHOOSE_SOURCES, ROUTES.OVERVIEW]);
      });
    });
  });
});
