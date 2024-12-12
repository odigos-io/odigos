// const input = {
//   name: {
//     name: 'Name',
//     value: 'load-generator',
//     status: null,
//     explain: '...',
//   },
// };

// const output = {
//   'name.name': 'Name',
//   'name.value': 'load-generator',
//   'name.status': null,
//   'name.explain': '...',
// };

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
