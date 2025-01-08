import { BUTTONS, DATA_IDS, ROUTES, SELECTED_ENTITIES } from '../constants';

describe('Onboarding', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it('Should contain a "default" namespace', () => {
    cy.visit(ROUTES.CHOOSE_SOURCES);
    cy.wait('@gql').then(() => {
      cy.get(DATA_IDS.SELECT_NAMESPACE).contains(SELECTED_ENTITIES.NAMESPACE).should('exist');
    });
  });

  it('Should contain a "Jaeger" destination, and it should be autocompleted', () => {
    cy.visit(ROUTES.CHOOSE_DESTINATION);
    cy.contains('button', BUTTONS.ADD_DESTINATION).click();
    cy.wait('@gql').then(() => {
      cy.get(DATA_IDS.SELECT_DESTINATION).contains(SELECTED_ENTITIES.DESTINATION_DISPLAY_NAME).should('exist').click();
      cy.get(DATA_IDS.SELECT_DESTINATION_AUTOFILL_FIELD).should('have.value', SELECTED_ENTITIES.DESTINATION_AUTOFILL_VALUE);
    });
  });

  it('Should allow the user to pass every step, and end-up on the "overview" page.', () => {
    cy.visit(ROUTES.CHOOSE_SOURCES);
    cy.contains('button', BUTTONS.NEXT).click();
    cy.location('pathname').should('eq', ROUTES.CHOOSE_DESTINATION);
    cy.contains('button', BUTTONS.DONE).click();
    cy.location('pathname').should('eq', ROUTES.OVERVIEW);
  });
});
