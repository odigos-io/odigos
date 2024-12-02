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

export const timeAgo = (timestamp: string) => {
  const now = new Date();
  const notificationTime = new Date(timestamp);

  if (isNaN(notificationTime.getTime())) {
    return '';
  }

  const differenceInMs = now.getTime() - notificationTime.getTime();
  const differenceInMinutes = Math.floor(differenceInMs / (1000 * 60));
  const differenceInHours = Math.floor(differenceInMinutes / 60);

  if (differenceInMinutes < 60) {
    if (differenceInMinutes === 0) {
      return 'Just now';
    } else if (differenceInMinutes === 1) {
      return '1 minute ago';
    }

    return `${differenceInMinutes} minutes ago`;
  } else if (differenceInHours < 24) {
    return `${differenceInHours} hours ago`;
  } else {
    const days = Math.floor(differenceInHours / 24);
    return `${days} days ago`;
  }
};

export function formatDate(dateString: string) {
  // Parse the date string into a Date object
  const date = new Date(dateString);

  // Get individual components
  const year = date.getUTCFullYear();
  const month = date.getUTCMonth(); // Note: months are zero-based
  const day = date.getUTCDate();
  const hours = date.getUTCHours();
  const minutes = date.getUTCMinutes();
  const seconds = date.getUTCSeconds();

  // Define month names
  const monthNames = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'];

  // Format the components into a readable string
  const formattedDate = `${monthNames[month]} ${day}, ${year} ${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;

  return formattedDate;
}
