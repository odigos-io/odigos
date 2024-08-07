describe('Overview Page Tests', () => {

    beforeEach(() => {
        cy.visit('localhost:3000')
    });

    it('should overview page redirect correctly', () => {
        cy.url().should('contain', `${Cypress.config('baseUrl')}/overview`);
    })

    it('should Sources exists', () => {
        cy.assertDemoAppsExistOverviewPage()
    })

    it('should Destinations exists', () => {
        cy.get('[data-id="destination-0"]').should('have.text', 'e2e-testsTempo')
    })

})
