import { BUTTONS, DATA_IDS, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

describe('Onboarding', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it('Should contain a "default" namespace, and it should have 5 sources', () => {
    cy.visit(ROUTES.CHOOSE_SOURCES);
    // Wait for the page to load
    cy.wait(1000).then(() => {
      // Wait for the namespaces to load
      cy.wait('@gql').then(() => {
        cy.get(DATA_IDS.SELECT_NAMESPACE).contains(SELECTED_ENTITIES.NAMESPACE).should('exist').click();
        // Wait for the sources to load
        cy.wait('@gql').then(() => {
          SELECTED_ENTITIES.NAMESPACE_SOURCES.forEach((sourceName) => {
            cy.get(DATA_IDS.SELECT_NAMESPACE).get(DATA_IDS.SELECT_SOURCE(sourceName)).contains(sourceName).should('exist');
          });
        });
      });
    });
  });

  it('Should contain a "Jaeger" destination, and it should be autocompleted', () => {
    cy.visit(ROUTES.CHOOSE_DESTINATION);
    // Wait for the page to load
    cy.wait(1000).then(() => {
      cy.contains('button', BUTTONS.ADD_DESTINATION).click();
      // Wait for the destinations to load
      cy.wait('@gql').then(() => {
        cy.get(DATA_IDS.SELECT_DESTINATION).contains(SELECTED_ENTITIES.DESTINATION_DISPLAY_NAME).should('exist').click();
        cy.get(DATA_IDS.SELECT_DESTINATION_AUTOFILL_FIELD).should('have.value', SELECTED_ENTITIES.DESTINATION_AUTOFILL_VALUE);
      });
    });
  });

  it('Should allow the user to pass every step, and end-up on the "overview" page.', () => {
    cy.visit(ROUTES.CHOOSE_SOURCES);
    // Wait for the page to load
    cy.wait(1000).then(() => {
      cy.contains('button', BUTTONS.BACK).should('not.exist');
      cy.contains('button', BUTTONS.NEXT).click();
      cy.location('pathname').should('eq', ROUTES.CHOOSE_DESTINATION);
      cy.contains(TEXTS.NO_SOURCES_SELECTED).should('exist');
      cy.contains('button', BUTTONS.BACK).should('exist');
      cy.contains('button', BUTTONS.DONE).click();
      cy.location('pathname').should('eq', ROUTES.OVERVIEW);
    });
  });
});
