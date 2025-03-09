export const deepClone: <T = any>(item: T) => T = (item) => {
  return JSON.parse(JSON.stringify(item));
};
