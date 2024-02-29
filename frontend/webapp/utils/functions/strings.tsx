export function capitalizeFirstLetter(string: string) {
  return string.charAt(0).toUpperCase() + string.slice(1);
}

export function safeJsonParse<T>(str: string | undefined, fallback: T): T {
  if (!str) return fallback;
  try {
    const parsed = JSON.parse(str) as T;
    return parsed;
  } catch (e) {
    console.error('Error parsing JSON string:', e);
    return fallback;
  }
}

export function cleanObjectEmptyStringsValues(
  obj: Record<string, string>
): Record<string, string> {
  const cleanArray = (arr: any[]) => arr.filter((item) => item !== '');
  const cleanObject = (o: Record<string, any>) =>
    Object.fromEntries(
      Object.entries(o).filter(([key, value]) => key !== '' && value !== '')
    );

  return Object.entries(obj).reduce((acc, [key, value]) => {
    try {
      const parsed = JSON.parse(value);
      if (Array.isArray(parsed)) {
        acc[key] = JSON.stringify(cleanArray(parsed));
      } else if (typeof parsed === 'object' && parsed !== null) {
        acc[key] = JSON.stringify(cleanObject(parsed));
      } else {
        acc[key] = value;
      }
    } catch (error) {
      // In case JSON.parse fails, assume value is a plain string
      acc[key] = value;
    }
    return acc;
  }, {} as Record<string, string>);
}
