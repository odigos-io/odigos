import { safeJsonParse } from '@/utils';
import type { ActionDataParsed, ActionInput } from '@/types';

const buildDrawerItem = (id: string, formData: ActionInput): ActionDataParsed => {
  const { type, name, notes, signals, disable, details } = formData;

  return {
    id,
    type,
    spec: {
      actionName: name,
      notes: notes,
      signals: signals,
      disabled: disable,
      ...safeJsonParse(details, {}),
    },
  };
};

export default buildDrawerItem;
