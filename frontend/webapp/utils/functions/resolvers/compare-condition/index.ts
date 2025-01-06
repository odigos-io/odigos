export const compareCondition = (renderCondition: string[], fields: { name: string; value: any }[]) => {
  if (!renderCondition || !renderCondition.length) return true;
  if (renderCondition.length === 1) return renderCondition[0] == 'true';

  const [key, cond, val] = renderCondition;
  const field = fields.find((field) => field.name === key);

  if (!field) {
    console.warn(`Field with name ${key} not found, condition will be skipped`);
    return true;
  }

  switch (cond) {
    case '===':
    case '==':
      return field.value === val;
    case '!==':
    case '!=':
      return field.value !== val;
    case '>':
      return field.value > val;
    case '<':
      return field.value < val;
    case '>=':
      return field.value >= val;
    case '<=':
      return field.value <= val;
    default:
      console.warn(`Invalid condition ${cond}, condition will be skipped`);
      return true;
  }
};
