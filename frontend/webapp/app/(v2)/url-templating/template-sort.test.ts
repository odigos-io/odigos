import { sortUrlTemplates } from './template-sort';

describe('sortUrlTemplates', () => {
  it('orders paths lexicographically with numeric segments', () => {
    expect(
      sortUrlTemplates(['/users/{id}/orders', '/api/v2/items', '/api/v10/items', '/accounts']),
    ).toEqual(['/accounts', '/api/v10/items', '/api/v2/items', '/users/{id}/orders']);
  });
});
