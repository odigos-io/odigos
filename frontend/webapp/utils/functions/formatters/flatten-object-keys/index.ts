/**
 * Recursively flattens a nested object into a single-level object where each key
 * represents the path to its corresponding value in the original object. Keys for nested
 * properties are concatenated using a dot (`.`) as a separator, while array elements
 * include their index in square brackets (`[]`).
 *
 * @param {Record<string, any>} obj - The input object to be flattened.
 * @param {string} [prefix=''] - The current prefix for the keys, used for recursion.
 * @param {Record<string, any>} [result={}] - The accumulator object that stores the flattened result.
 * @returns {Record<string, any>} A new object where all nested properties are flattened into
 *                                a single level with their paths as keys.
 *
 * @example
 * const input = {
 *   name: {
 *     name: 'Name',
 *     value: 'load-generator',
 *     status: null,
 *     explain: '...',
 *   },
 * };
 *
 * const output = flattenObjectKeys(input);
 * Output:
 * {
 *   'name.name': 'Name',
 *   'name.value': 'load-generator',
 *   'name.status': null,
 *   'name.explain': '...',
 * }
 */

export const flattenObjectKeys = (obj: Record<string, any>, prefix: string = '', result: Record<string, any> = {}) => {
  for (const key in obj) {
    if (obj.hasOwnProperty(key)) {
      const value = obj[key];
      const newKey = prefix ? `${prefix}.${key}` : key;

      if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
        // Recurse for nested objects
        flattenObjectKeys(value, newKey, result);
      } else if (Array.isArray(value)) {
        value.forEach((item, index) => {
          const arrayKey = `${newKey}[${index}]`;

          if (item !== null && typeof item === 'object') {
            // Recurse for objects in arrays
            flattenObjectKeys(item, arrayKey, result);
          } else {
            // Assign primitive array values
            result[arrayKey] = item;
          }
        });
      } else {
        // Assign non-object, non-array values
        result[newKey] = value;
      }
    }
  }

  return result;
};
