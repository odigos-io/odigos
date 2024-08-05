describe('Basic UI Tests', () => {
    it('Main page loads', () => {
        cy.visit('localhost:3000')
        cy.get('[data-id="namespace-0"]').should('have.text', 'defaultcouponnn')
        cy.get('[data-id="destination-0"]').should('have.text', 'FDSAElasticsearchnfgfsdg')
    })

})
