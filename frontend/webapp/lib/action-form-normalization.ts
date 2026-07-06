type RenameRow = {
  key?: unknown;
  value?: unknown;
  oldKey?: unknown;
  newKey?: unknown;
};

export type RenamesValue = Record<string, unknown> | RenameRow[] | string | null | undefined;

const asString = (value: unknown): string => {
  if (typeof value === 'string') return value;
  if (value == null) return '';
  return String(value);
};

const normalizeRenamesRows = (rows: RenameRow[]): Record<string, string> => {
  return rows.reduce<Record<string, string>>((acc, row) => {
    const oldKey = asString(row.key ?? row.oldKey);
    if (!oldKey) return acc;

    acc[oldKey] = asString(row.value ?? row.newKey);
    return acc;
  }, {});
};

const normalizeRenamesObject = (renames: Record<string, unknown>): Record<string, string> => {
  return Object.entries(renames).reduce<Record<string, string>>((acc, [oldKey, newKey]) => {
    if (!oldKey) return acc;

    acc[oldKey] = asString(newKey);
    return acc;
  }, {});
};

export const normalizeActionRenames = (renames: RenamesValue): Record<string, string> => {
  if (!renames) return {};

  if (typeof renames === 'string') {
    try {
      return normalizeActionRenames(JSON.parse(renames));
    } catch {
      return {};
    }
  }

  if (Array.isArray(renames)) {
    return normalizeRenamesRows(renames);
  }

  if (typeof renames === 'object') {
    return normalizeRenamesObject(renames);
  }

  return {};
};
