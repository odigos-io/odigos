export function stringifyNonStringValues(obj: Record<string, any>): Record<string, string> {
  return Object.entries(obj).reduce((acc, [key, value]) => {
    // Check if the value is already a string
    if (typeof value === 'string') {
      acc[key] = value;
    } else {
      // If not, stringify the value
      acc[key] = JSON.stringify(value);
    }
    return acc;
  }, {} as Record<string, string>);
}
