import type { ActualDestination, DestinationInput } from '@/types';

const buildDrawerItem = (id: string, formData: DestinationInput, drawerItem: ActualDestination): ActualDestination => {
  const { name, exportedSignals, fields } = formData;
  const { destinationType, conditions } = drawerItem || {};

  let fieldsStringified: string | Record<string, any> = {};
  fields.forEach(({ key, value }) => (fieldsStringified[key] = value));
  fieldsStringified = JSON.stringify(fieldsStringified);

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
