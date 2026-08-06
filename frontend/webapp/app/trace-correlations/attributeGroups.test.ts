import { groupAttributesByPrefix } from './attributeGroups';

describe('groupAttributesByPrefix', () => {
  it('groups attributes by namespace prefix', () => {
    expect(
      groupAttributesByPrefix([
        'http.route',
        'rpc.service',
        'db.system',
        'http.method',
        'server.address',
      ]),
    ).toEqual([
      { prefix: 'db', label: 'db.', values: ['db.system'] },
      { prefix: 'http', label: 'http.', values: ['http.method', 'http.route'] },
      { prefix: 'rpc', label: 'rpc.', values: ['rpc.service'] },
      { prefix: 'server', label: 'server.', values: ['server.address'] },
    ]);
  });
});
