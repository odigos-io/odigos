import { safeJsonParse } from '@/utils';
import { DestinationDetailsField, DynamicField } from '@/types';

export function useConnectDestinationForm() {
  function buildFormDynamicFields(
    fields: DestinationDetailsField[]
  ): DynamicField[] {
    return fields
      .map((field) => {
        const { name, componentType, displayName, componentProperties } = field;

        let componentPropertiesJson;
        switch (componentType) {
          case 'dropdown':
            componentPropertiesJson = safeJsonParse<{ [key: string]: string }>(
              componentProperties,
              {}
            );

            const options = Object.entries(componentPropertiesJson.values).map(
              ([key, value]) => ({
                id: key,
                value,
              })
            );

            return {
              name,
              componentType,
              title: displayName,
              onSelect: () => {},
              options,
              selectedOption: options[0],
              ...componentPropertiesJson,
            };

          case 'input':
          case 'textarea':
            componentPropertiesJson = safeJsonParse<string[]>(
              componentProperties,
              []
            );
            return {
              name,
              componentType,
              title: displayName,
              ...componentPropertiesJson,
            };

          case 'multiInput':
            componentPropertiesJson = safeJsonParse<string[]>(
              componentProperties,
              []
            );
            return {
              name,
              componentType,
              title: displayName,
              ...componentPropertiesJson,
            };
          default:
            return undefined;
        }
      })
      .filter((field): field is DynamicField => field !== undefined);
  }

  return { buildFormDynamicFields };
}
