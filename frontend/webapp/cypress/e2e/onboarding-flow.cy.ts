describe('On Boarding Flow Tests', () => {

        beforeEach(() => {
            cy.visit('localhost:3000')
        });

        it('Source page redirects correctly', () => {
            cy.url().should('eq', 'http://localhost:3000/choose-sources');
        })

        it('Run on the onboarding flow', () => {
            // Select All Apps
            cy.get('[data-cy="choose-source-coupon"]').should('exist').click()
            cy.get('[data-cy="choose-source-frontend"]').should('exist').click()
            cy.get('[data-cy="choose-source-inventory"]').should('exist').click()
            cy.get('[data-cy="choose-source-membership"]').should('exist').click()
            cy.get('[data-cy="choose-source-pricing"]').should('exist').click()

            // Click Next Page
            cy.get('[data-cy="choose-source-next-click"]').should('exist').click()

            // Select Tempo
            cy.url().should('eq', 'http://localhost:3000/choose-destination');
            cy.get('[data-cy="choose-destination-Tempo"]').should('exist').click()

            // Fill Destination Form
            cy.url().should('eq', 'http://localhost:3000/connect-destination?type=tempo');
            cy.get('[data-cy=create-destination-input-name]').type('e2e-tests');
            cy.get('[data-cy=create-destination-input-endpoint]').type('e2e-tests-tempo.traces:4317');
            cy.get('[data-cy="create-destination-create-click"]').should('exist').click()

            cy.url().should('eq', 'http://localhost:3000/overview');

        });
    }
);
