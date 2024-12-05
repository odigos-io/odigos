export function cleanObjectEmptyStringsValues(obj: Record<string, any>): Record<string, any> {
  const cleanArray = (arr: any[]): any[] =>
    arr.filter((item) => {
      if (typeof item === 'object' && item !== null) {
        return item.key !== '' && item.value !== '';
      }
      return item !== '';
    });

  const cleanObject = (o: Record<string, any>): Record<string, any> =>
    Object.fromEntries(
      Object.entries(o)
        .filter(([key, value]) => key !== '' && value !== '')
        .map(([key, value]) => {
          if (Array.isArray(value)) return [key, cleanArray(value)];
          else if (typeof value === 'object' && value !== null) return [key, cleanObject(value)];
          return [key, value];
        }),
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
      // Handle non-stringified objects or arrays directly
      if (typeof value === 'object' && value !== null) {
        if (Array.isArray(value)) acc[key] = JSON.stringify(cleanArray(value));
        else acc[key] = JSON.stringify(cleanObject(value));
      } else {
        // In case JSON.parse fails, assume value is a plain string or non-object/array
        acc[key] = value;
      }
    }
    return acc;
  }, {} as Record<string, any>);
}
