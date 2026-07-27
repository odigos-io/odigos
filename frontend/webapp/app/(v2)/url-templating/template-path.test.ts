import { parseTemplatePath } from './template-path';

describe('parseTemplatePath', () => {
  it('classifies static, variable, and wildcard segments', () => {
    expect(parseTemplatePath('/users/{id}/orders/{orderId}/items/*')).toEqual([
      { kind: 'static', text: 'users' },
      { kind: 'variable', text: '{id}' },
      { kind: 'static', text: 'orders' },
      { kind: 'variable', text: '{orderId}' },
      { kind: 'static', text: 'items' },
      { kind: 'wildcard', text: '*' },
    ]);
  });

  it('handles templates without a leading slash', () => {
    expect(parseTemplatePath('health')).toEqual([{ kind: 'static', text: 'health' }]);
  });
});
