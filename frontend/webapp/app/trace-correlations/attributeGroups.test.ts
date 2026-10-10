import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { groupAttributesByPrefix } from './attributeGroups';

describe('groupAttributesByPrefix', () => {
  it('groups attributes by namespace prefix', () => {
    assert.deepEqual(
      groupAttributesByPrefix([
        'http.route',
        'rpc.service',
        'db.system',
        'http.method',
        'server.address',
      ]),
      [
        { prefix: 'db', label: 'db.', values: ['db.system'] },
        { prefix: 'http', label: 'http.', values: ['http.method', 'http.route'] },
        { prefix: 'rpc', label: 'rpc.', values: ['rpc.service'] },
        { prefix: 'server', label: 'server.', values: ['server.address'] },
      ],
    );
  });
});
