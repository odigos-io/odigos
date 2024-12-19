export const removeEmptyValuesFromObject = (obj: Record<string, any>): Record<string, any> => {
  if (typeof obj !== 'object') return obj;

  const result: Record<string, any> = Array.isArray(obj) ? [] : {};

  Object.keys(obj).forEach((key) => {
    const value = obj[key];

    if (Array.isArray(value)) {
      // Remove empty arrays or recursively clean non-empty ones
      const filteredArray = value.filter((item) => item !== null && item !== undefined && item !== '');
      if (filteredArray.length > 0) result[key] = filteredArray.map((item) => removeEmptyValuesFromObject(item));
    } else if (typeof value === 'object' && value !== null) {
      // Recursively clean nested objects
      const nestedObject = removeEmptyValuesFromObject(value);
      if (Object.keys(nestedObject).length > 0) result[key] = nestedObject;
    } else if (![undefined, null, ''].includes(value)) {
      // Keep valid values
      result[key] = value;
    }
  });

  return result;
};
