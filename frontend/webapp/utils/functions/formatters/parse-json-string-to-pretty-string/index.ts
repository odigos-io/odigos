export const parseJsonStringToPrettyString = (value: string) => {
  let str = '';

  try {
    const parsed = JSON.parse(value);

    // Handle arrays
    if (Array.isArray(parsed)) {
      str = parsed
        .map((item) => {
          if (typeof item === 'object' && item !== null) return `${item.key}: ${item.value}`;
          else return item;
        })
        .join(', ');
    }

    // Handle objects (non-array JSON objects)
    else if (typeof parsed === 'object' && parsed !== null) {
      str = Object.entries(parsed)
        .map(([key, val]) => `${key}: ${val}`)
        .join(', ');
    }

    // Should never reach this if it's a string (it will throw)
    else {
      str = value;
    }
  } catch (error) {
    str = value;
  }

  return str;
};
