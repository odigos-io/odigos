type ActionLike = {
  fields?: Record<string, unknown>;
};

type ActionMutationVarsLike = {
  action?: ActionLike;
};

export const parseRenamesString = (value: string): Record<string, string> => {
  try {
    const parsed = JSON.parse(value);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? (parsed as Record<string, string>) : {};
  } catch {
    return {};
  }
};

export const normalizeRenamesForWire = (renames: unknown): string | null => {
  if (!renames) return null;
  if (typeof renames === 'string') return renames.trim() ? renames : null;

  const normalized: Record<string, string> = {};

  if (Array.isArray(renames)) {
    renames.forEach((row) => {
      if (!row || typeof row !== 'object') return;
      const { key, value } = row as { key?: unknown; value?: unknown };
      if (typeof key === 'string' && key && typeof value === 'string') {
        normalized[key] = value;
      }
    });
  } else if (typeof renames === 'object') {
    Object.entries(renames as Record<string, unknown>).forEach(([key, value]) => {
      if (key && typeof value === 'string') {
        normalized[key] = value;
      }
    });
  }

  return Object.keys(normalized).length ? JSON.stringify(normalized) : null;
};

export const sanitizeExtractAttributeForWire = (fields: Record<string, unknown>): Record<string, unknown> => {
  const extractAttribute = fields.extractAttribute as { extractions?: unknown[] } | null | undefined;
  if (!extractAttribute || !Array.isArray(extractAttribute.extractions)) return fields;

  return {
    ...fields,
    extractAttribute: {
      ...extractAttribute,
      extractions: extractAttribute.extractions.map((extraction) => {
        if (!extraction || typeof extraction !== 'object') return extraction;

        const next: Record<string, unknown> = { ...(extraction as Record<string, unknown>) };
        delete next.method;
        if (!next.lookupKey) delete next.lookupKey;
        if (!next.regex) delete next.regex;
        if (!next.dataFormat) delete next.dataFormat;
        return next;
      }),
    },
  };
};

export const normalizeActionForWire = (vars: ActionMutationVarsLike): Record<string, unknown> => {
  const { action } = vars;
  const fields = action?.fields ?? {};
  const normalizedFields = sanitizeExtractAttributeForWire({
    ...fields,
    // Mirror the legacy hook: an empty/missing map is sent as null so the
    // controller skips the RenameAttribute config entirely.
    renames: normalizeRenamesForWire(fields.renames),
  });

  return {
    ...vars,
    action: {
      ...action,
      fields: normalizedFields,
    },
  };
};
