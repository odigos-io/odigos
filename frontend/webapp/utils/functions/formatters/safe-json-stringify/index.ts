export const safeJsonStringify = (obj?: Record<any, any>, indent = 2) => {
  return JSON.stringify(obj || {}, null, indent);
};
