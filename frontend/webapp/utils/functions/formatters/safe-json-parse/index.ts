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
