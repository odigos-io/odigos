import { visitPage } from '../functions';
import { BUTTONS, DATA_IDS, ROUTES, SELECTED_ENTITIES, TEXTS } from '../constants';

describe('Onboarding', () => {
  beforeEach(() => cy.intercept('/graphql').as('gql'));

  it(`Should contain a "${SELECTED_ENTITIES.NAMESPACE}" namespace, and it should have 5 sources`, () => {
    visitPage(ROUTES.CHOOSE_SOURCES, () => {
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

  it(`Should contain a "${SELECTED_ENTITIES.DESTINATION.TYPE}" destination, and it should be autocompleted`, () => {
    visitPage(ROUTES.CHOOSE_DESTINATION, () => {
      cy.contains('button', BUTTONS.ADD_DESTINATION).click();
      // Wait for the destinations to load
      cy.wait('@gql').then(() => {
        cy.get(DATA_IDS.SELECT_DESTINATION).contains(SELECTED_ENTITIES.DESTINATION.DISPLAY_NAME).should('exist').click();
        cy.get(DATA_IDS.SELECT_DESTINATION_AUTOFILL_FIELD).should('have.value', SELECTED_ENTITIES.DESTINATION.AUTOFILL_VALUE);
      });
    });
  });

  it('Should allow the user to pass every step, and end-up on the "overview" page.', () => {
    visitPage(ROUTES.CHOOSE_STREAM, () => {
      cy.contains('button', BUTTONS.BACK).should('not.exist');
      cy.contains('button', BUTTONS.NEXT).should('exist').click();

      cy.location('pathname').should('eq', ROUTES.CHOOSE_SOURCES);
      cy.contains('button', BUTTONS.BACK).should('exist');
      cy.contains('button', BUTTONS.NEXT).should('exist').click();

      cy.location('pathname').should('eq', ROUTES.CHOOSE_DESTINATION);
      cy.contains(TEXTS.NO_SOURCES_SELECTED).should('exist');
      cy.contains('button', BUTTONS.BACK).should('exist');
      cy.contains('button', BUTTONS.NEXT).should('exist').click();

      cy.location('pathname').should('eq', ROUTES.SETUP_SUMMARY);
      cy.contains('button', BUTTONS.BACK).should('exist');
      cy.contains('button', BUTTONS.DONE).should('exist').click();

      cy.location('pathname').should('eq', ROUTES.OVERVIEW);
    });
  });
});
