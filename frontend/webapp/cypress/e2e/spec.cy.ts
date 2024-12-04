describe('template spec', () => {
  it('passes', () => {
    cy.visit('/');

    expect(true).to.equal(true);
  });
});
