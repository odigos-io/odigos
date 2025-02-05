import { type DestinationInput } from '@/types';
import { type Destination } from '@odigos/ui-containers';

const buildDrawerItem = (id: string, formData: DestinationInput, drawerItem: Destination): Destination => {
  const { name, exportedSignals, fields } = formData;
  const { destinationType, conditions } = drawerItem || {};

  const fieldsObject: Record<string, any> = {};
  fields.forEach(({ key, value }) => {
    fieldsObject[key] = value;
  });

  const fieldsStringified = JSON.stringify(fieldsObject);

  return {
    id,
    name,
    exportedSignals,
    fields: fieldsStringified,
    destinationType,
    conditions,
  };
};

export default buildDrawerItem;
