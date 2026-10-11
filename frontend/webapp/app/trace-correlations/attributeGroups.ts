export type AttributePrefixGroup = {
  prefix: string;
  label: string;
  values: string[];
};

export function getAttributePrefix(value: string) {
  const dotIndex = value.indexOf('.');
  if (dotIndex <= 0) {
    return 'other';
  }
  return value.slice(0, dotIndex);
}

export function groupAttributesByPrefix(values: string[]): AttributePrefixGroup[] {
  const groups = new Map<string, string[]>();

  for (const value of values) {
    const prefix = getAttributePrefix(value);
    const existing = groups.get(prefix) ?? [];
    existing.push(value);
    groups.set(prefix, existing);
  }

  return Array.from(groups.entries())
    .sort(([left], [right]) => {
      if (left === 'other') {
        return 1;
      }
      if (right === 'other') {
        return -1;
      }
      return left.localeCompare(right);
    })
    .map(([prefix, groupValues]) => ({
      prefix,
      label: prefix === 'other' ? 'other' : `${prefix}.`,
      values: [...groupValues].sort((left, right) => left.localeCompare(right)),
    }));
}
