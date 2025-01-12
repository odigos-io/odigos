// Sometimes we need to allow "zero" values, and a simple "!val" check would result in false positives.
// This function is a strict check for empty values, permitting values like "0" and "false".

export const isEmpty = (val: any) => {
  if (Array.isArray(val)) {
    return !val.length;
  } else if (typeof val === 'object') {
    return !Object.keys(val).length;
  } else {
    return [undefined, null, ''].includes(val);
  }
};
