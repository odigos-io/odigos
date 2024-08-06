describe('Overview Page Tests', () => {

    beforeEach(() => {
        cy.visit('localhost:3000')
    });

    it('should overview page redirect correctly', () => {
        cy.url().should('eq', 'http://localhost:3000/overview');
    })

    it('should Sources exists', () => {
        cy.get('[data-id="namespace-0"]').should('have.text', 'defaultcoupon')
        cy.get('[data-id="namespace-1"]').should('have.text', 'defaultfrontend')
        cy.get('[data-id="namespace-2"]').should('have.text', 'defaultinventory')
        cy.get('[data-id="namespace-3"]').should('have.text', 'defaultmembership')
        cy.get('[data-id="namespace-4"]').should('have.text', 'defaultpricing')
    })

    it('should Destinations exists', () => {
        cy.get('[data-id="destination-0"]').should('have.text', 'e2e-testsTempo')
    })

})
