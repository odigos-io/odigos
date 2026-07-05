type RenameRow = {
  key?: unknown;
  value?: unknown;
};

const isPlainObject = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === 'object' && !Array.isArray(value);

const normalizeRenameRows = (renames: RenameRow[]): Record<string, string> => {
  return renames.reduce<Record<string, string>>((acc, row) => {
    if (typeof row.key === 'string' && row.key) {
      acc[row.key] = typeof row.value === 'string' ? row.value : '';
    }
    return acc;
  }, {});
};

export const normalizeActionRenames = (renames: unknown): Record<string, string> => {
  if (!renames) return {};

  if (typeof renames === 'string') {
    try {
      return normalizeActionRenames(JSON.parse(renames));
    } catch {
      return {};
    }
  }

  if (Array.isArray(renames)) {
    return normalizeRenameRows(renames as RenameRow[]);
  }

  if (isPlainObject(renames)) {
    return Object.entries(renames).reduce<Record<string, string>>((acc, [key, value]) => {
      if (typeof value === 'string') {
        acc[key] = value;
      }
      return acc;
    }, {});
  }

  return {};
};
