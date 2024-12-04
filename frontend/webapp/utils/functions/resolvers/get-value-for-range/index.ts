export const getValueForRange = (current: number, matrix: [number, number | null, any][]) => {
  // CURRENT: represents the current value (such as window width)
  // MATRIX: represents the ranges (index[0] == min, index[1] == max, index[2] == value to get)

  // EXAMPLE:
  // getValueForRange(width, [
  //   [0, 1000, 'small'], // ---> from 0 to 1000, return "small"
  //   [1000, null, 'big'], // ---> from 1000 to infinite, return "big"
  // ])

  const found = matrix.find(([min, max]) => current >= min && (max === null || current <= max));

  return found?.[2] || null;
};
