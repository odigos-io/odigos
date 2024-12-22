import { safeJsonParse, INPUT_TYPES } from '@/utils';
import { DestinationDetailsField, DynamicField } from '@/types';

export function useConnectDestinationForm() {
  function buildFormDynamicFields(fields: DestinationDetailsField[]): DynamicField[] {
    return fields
      .map((field) => {
        const { componentType, displayName, initialValue, componentProperties, ...restOfField } = field;

        let componentPropertiesJson;
        let initialValuesJson;

        switch (componentType) {
          case INPUT_TYPES.INPUT:
          case INPUT_TYPES.TEXTAREA:
          case INPUT_TYPES.CHECKBOX:
          case INPUT_TYPES.KEY_VALUE_PAIR:
            componentPropertiesJson = safeJsonParse<{ [key: string]: string }>(componentProperties, {});

            return {
              componentType,
              title: displayName,
              value: initialValue,
              ...restOfField,
              ...componentPropertiesJson,
            };

          case INPUT_TYPES.MULTI_INPUT:
            componentPropertiesJson = safeJsonParse<{ [key: string]: string }>(componentProperties, {});
            initialValuesJson = safeJsonParse<string[]>(initialValue, []);

            return {
              componentType,
              title: displayName,
              value: initialValuesJson,
              initialValues: initialValuesJson,
              ...restOfField,
              ...componentPropertiesJson,
            };

          case INPUT_TYPES.DROPDOWN:
            componentPropertiesJson = safeJsonParse<{ [key: string]: string }>(componentProperties, {});

            const options = Array.isArray(componentPropertiesJson.values)
              ? componentPropertiesJson.values.map((value) => ({
                  id: value,
                  value,
                }))
              : Object.entries(componentPropertiesJson.values).map(([key, value]) => ({
                  id: key,
                  value,
                }));

            return {
              componentType,
              title: displayName,
              value: initialValue,
              placeholder: componentPropertiesJson.placeholder || 'Select an option',
              options,
              onSelect: () => {},
              ...restOfField,
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
