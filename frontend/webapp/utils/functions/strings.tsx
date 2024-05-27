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
  obj: Record<string, any>
): Record<string, any> {
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
          else if (typeof value === 'object' && value !== null)
            return [key, cleanObject(value)];
          return [key, value];
        })
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
export function stringifyNonStringValues(
  obj: Record<string, any>
): Record<string, string> {
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
    return `${differenceInMinutes} minutes ago`;
  } else if (differenceInHours < 24) {
    return `${differenceInHours} hours ago`;
  } else {
    const days = Math.floor(differenceInHours / 24);
    return `${days} days ago`;
  }
};
