import { safeJsonParse } from '@/utils';
import type { ActionDataParsed, ActionInput } from '@/types';

const buildDrawerItem = (id: string, formData: ActionInput, drawerItem: ActionDataParsed): ActionDataParsed => {
  const { type, name, notes, signals, disable, details } = formData;
  const { status } = drawerItem;

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
    status,
  };
};

export default buildDrawerItem;
