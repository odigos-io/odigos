export function safeJsonParse<T>(val: string | Record<any, any> | undefined, fallback: T): T {
  if (!val) return fallback;
  if (typeof val === 'object') return val;

  try {
    const parsed = JSON.parse(val) as T;
    return parsed;
  } catch (e) {
    console.error('Error parsing JSON string:', e);
    return fallback;
  }
}
