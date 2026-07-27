import { extractUrlPath, simulateCustomTemplate } from './template-simulator';

describe('extractUrlPath', () => {
  it('extracts pathname from http URLs with query strings', () => {
    expect(extractUrlPath('http://foo.bar/bla/xxx/yyy/1234?adsfds=fsadfds')).toEqual({
      segments: ['bla', 'xxx', 'yyy', '1234'],
      hadLeadingSlash: true,
    });
  });

  it('extracts pathname from https URLs with hash fragments', () => {
    expect(extractUrlPath('https://api.example.com/users/42#section')).toEqual({
      segments: ['users', '42'],
      hadLeadingSlash: true,
    });
  });

  it('handles bare paths with query strings', () => {
    expect(extractUrlPath('/users/42?tab=1')).toEqual({
      segments: ['users', '42'],
      hadLeadingSlash: true,
    });
  });
});

describe('simulateCustomTemplate', () => {
  it('templates variable segments when static parts match', () => {
    expect(simulateCustomTemplate('/users/{id}', '/users/42')).toEqual({
      status: 'matched',
      outputPath: '/users/{id}',
    });
  });

  it('accepts full http URLs with query strings', () => {
    expect(
      simulateCustomTemplate('/bla/xxx/yyy/{id}', 'http://foo.bar/bla/xxx/yyy/1234?adsfds=fsadfds'),
    ).toEqual({
      status: 'matched',
      outputPath: '/bla/xxx/yyy/{id}',
    });
  });

  it('accepts full URLs and uses pathname only', () => {
    expect(simulateCustomTemplate('/users/{id}', 'https://api.example.com/users/42?q=1')).toEqual({
      status: 'matched',
      outputPath: '/users/{id}',
    });
  });

  it('rejects segment count mismatch', () => {
    const result = simulateCustomTemplate('/users/{id}', '/users/42/orders/9');
    expect(result.status).toBe('no_match');
    if (result.status === 'no_match') {
      expect(result.message).toContain('expects 2');
    }
  });

  it('rejects static segment mismatch', () => {
    expect(simulateCustomTemplate('/users/{id}', '/customers/42').status).toBe('no_match');
  });

  it('preserves wildcard segment values in output', () => {
    expect(simulateCustomTemplate('/items/*', '/items/sale')).toEqual({
      status: 'matched',
      outputPath: '/items/sale',
    });
  });
});
