/** Sort key for URL template paths (stable, path-friendly lexicographic order). */
export function compareUrlTemplates(a: string, b: string): number {
  const left = a.trim();
  const right = b.trim();
  if (left === right) {
    return 0;
  }
  return left.localeCompare(right, undefined, { sensitivity: 'base', numeric: true });
}

export function sortUrlTemplates(templates: readonly string[]): string[] {
  return [...templates].sort(compareUrlTemplates);
}
