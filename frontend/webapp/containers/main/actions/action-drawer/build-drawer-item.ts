import { ActionDataParsed, ActionInput } from '@/types';
import { safeJsonParse } from '@/utils';

const buildDrawerItem = (id: string, { type, name, notes, signals, disable, details }: ActionInput): ActionDataParsed => {
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
