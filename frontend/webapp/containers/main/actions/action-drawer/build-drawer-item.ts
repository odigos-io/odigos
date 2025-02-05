import { type ActionInput } from '@/types';
import { safeJsonParse } from '@odigos/ui-utils';
import { type Action } from '@odigos/ui-containers';

const buildDrawerItem = (id: string, formData: ActionInput, drawerItem: Action): Action => {
  const { type, name, notes, signals, disable, details } = formData;
  const { conditions } = drawerItem;

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
    conditions,
  };
};

export default buildDrawerItem;
