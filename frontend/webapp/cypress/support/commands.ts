export {};

declare global {
    namespace Cypress {
        interface Chainable<Subject = any> {
            assertDemoAppsExistOverviewPage(): Chainable<any>;
        }
    }
}

Cypress.Commands.add('assertDemoAppsExistOverviewPage', () => {
    cy.get('[data-id="namespace-0"]').should('have.text', 'defaultcoupon')
    cy.get('[data-id="namespace-1"]').should('have.text', 'defaultfrontend')
    cy.get('[data-id="namespace-2"]').should('have.text', 'defaultinventory')
    cy.get('[data-id="namespace-3"]').should('have.text', 'defaultmembership')
    cy.get('[data-id="namespace-4"]').should('have.text', 'defaultpricing')
});