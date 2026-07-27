import { buildCustomTemplateTree } from './custom-template-tree';

describe('buildCustomTemplateTree', () => {
  it('groups templates by shared path segments', () => {
    const tree = buildCustomTemplateTree([
      {
        template: '/users/{id}',
        scopeTokens: [{ type: 'entire_cluster' }],
        count: 1,
        actionLabels: ['a'],
        actionId: 'action-a',
        actionDisplayName: null,
        actionDisabled: false,
        actionUiGenerated: false,
      },
      {
        template: '/users/{id}/orders',
        scopeTokens: [{ type: 'entire_cluster' }],
        count: 1,
        actionLabels: ['a'],
        actionId: 'action-a',
        actionDisplayName: null,
        actionDisabled: false,
        actionUiGenerated: false,
      },
      {
        template: '/health',
        scopeTokens: [{ type: 'entire_cluster' }],
        count: 1,
        actionLabels: ['a'],
        actionId: 'action-a',
        actionDisplayName: null,
        actionDisabled: false,
        actionUiGenerated: false,
      },
    ]);

    expect(tree.map((node) => node.segment)).toEqual(['health', 'users']);
    const users = tree.find((node) => node.segment === 'users');
    expect(users?.children.map((node) => node.segment)).toEqual(['{id}']);
    expect(users?.children[0]?.entries).toHaveLength(1);
    expect(users?.children[0]?.children[0]?.segment).toBe('orders');
  });
});
