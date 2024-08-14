describe('Action Addition Testing', () => {
    it('Add an Action', () => {
        const baseUrl = Cypress.config('baseUrl');

        cy.visit('/')
        cy.get('[data-cy="menu-Actions"]').should('exist').click()
        cy.url().should('eq', `${baseUrl}/actions`);

        cy.get('[data-cy="add-action-button"]').should('exist').click()

        cy.url().should('eq', `${baseUrl}/choose-action`);
        cy.get('[data-cy="choose-action-ProbabilisticSampler"]').should('exist').click()

        cy.url().should('eq', `${baseUrl}/create-action?type=ProbabilisticSampler`);
        cy.get('[data-cy="create-action-input-name"]').type('action-test');
        cy.get('[data-cy="create-action-sampling-percentage"]').type('0.5');
        cy.get('[data-cy="create-action-onclick"]').click()

        cy.url().should('eq', `${baseUrl}/actions`);
        cy.get('[data-cy="actions-action-name"]').should('have.text', 'action-test ')
    });
});