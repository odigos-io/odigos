import { safeJsonParse } from '@/utils';
import type { ActionDataParsed, ActionInput } from '@/types';

const buildDrawerItem = (id: string, formData: ActionInput, drawerItem: ActionDataParsed): ActionDataParsed => {
  const { type, name, notes, signals, disable, details } = formData;
  const {} = drawerItem;

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
