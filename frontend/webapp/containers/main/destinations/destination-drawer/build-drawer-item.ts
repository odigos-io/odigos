import type { ActualDestination, DestinationInput, DestinationTypeItem } from '@/types';

const buildDrawerItem = (id: string, formData: DestinationInput, extra: DestinationTypeItem): ActualDestination => {
  const { name, exportedSignals, fields } = formData;
  const { type, displayName, imageUrl, supportedSignals } = extra || {};

  let fieldsStringified: string | Record<string, any> = {};
  fields.forEach(({ key, value }) => (fieldsStringified[key] = value));
  fieldsStringified = JSON.stringify(fieldsStringified);

  return {
    id,
    name,
    exportedSignals,
    fields: fieldsStringified,
    destinationType: {
      type,
      displayName,
      imageUrl,
      supportedSignals,
    },

    // TODO: map "conditions" (maybe ??)
    conditions: [],
  };
};

export default buildDrawerItem;
