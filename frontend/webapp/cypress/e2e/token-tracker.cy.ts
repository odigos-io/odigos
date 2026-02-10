import type { TokenPayload } from '@odigos/ui-kit/types';
import { ROUTES } from '../constants';
import { aliasQuery, awaitToast, handleExceptions, hasOperationName, visitPage } from '../functions';

const mockTokenQuery = (expiresAt: number) => {
  cy.intercept('/graphql', (req) => {
    if (hasOperationName(req, 'GetTokens')) {
      aliasQuery(req, 'GetTokens');

      req.reply((res) => {
        // This is to mock the tokens response
        res.body.data = {
          computePlatform: {
            apiTokens: [
              {
                name: 'Mock Token',
                token: 'j.w.t',
                issuedAt: 0,
                expiresAt,
              },
            ],
          },
        };
      });
    }
  }).as('gql');

  handleExceptions();
};

describe('Token Tracker', () => {
  it('Should track a "valid" token, and not show any notifications', () => {
    // in 30 days
    mockTokenQuery(new Date().getTime() + 1000 * 60 * 60 * 24 * 30);

    visitPage(ROUTES.OVERVIEW, () => {
      cy.get('[data-id=token-status]').should('not.exist');
      cy.get('[data-id=system-drawer]').click(); // testing that the UI does not crash
    });
  });

  it('Should track an "expiring" token, and show a warning notification', () => {
    // in 24 hours
    mockTokenQuery(new Date().getTime() + 1000 * 60 * 60 * 24);

    visitPage(ROUTES.OVERVIEW, () => {
      cy.wait('@gql').then(() => {
        awaitToast({ message: 'The token is about to expire in 1 day.' });
        cy.wait(1000).then(() => {
          cy.get('[data-id=token-status]').should('contain.text', 'The token is about to expire in 1 day.');
          cy.get('[data-id=system-drawer]').click(); // testing that the UI does not crash
        });
      });
    });
  });

  it('Should track an "expired" token, and show an error notification', () => {
    // 24 hours ago
    mockTokenQuery(new Date().getTime() - 1000 * 60 * 60 * 24);

    visitPage(ROUTES.OVERVIEW, () => {
      cy.wait('@gql').then(() => {
        awaitToast({ message: 'The token has expired 1 day ago.' });
        cy.wait(1000).then(() => {
          cy.get('[data-id=token-status]').should('contain.text', 'The token has expired 1 day ago.');
          cy.get('[data-id=system-drawer]').click(); // testing that the UI does not crash
        });
      });
    });
  });
});
