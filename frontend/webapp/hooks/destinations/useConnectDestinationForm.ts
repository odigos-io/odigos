import { safeJsonParse, INPUT_TYPES } from '@/utils';
import { DestinationDetailsField, DynamicField } from '@/types';

export function useConnectDestinationForm() {
  function buildFormDynamicFields(
    fields: DestinationDetailsField[]
  ): DynamicField[] {
    return fields
      .map((field) => {
        const {
          name,
          componentType,
          displayName,
          componentProperties,
          initialValue,
        } = field;

        let componentPropertiesJson;
        let initialValuesJson;
        switch (componentType) {
          case INPUT_TYPES.DROPDOWN:
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
              placeholder: 'Select an option',
              ...componentPropertiesJson,
            };

          case INPUT_TYPES.INPUT:
          case INPUT_TYPES.TEXTAREA:
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

          case INPUT_TYPES.MULTI_INPUT:
            componentPropertiesJson = safeJsonParse<string[]>(
              componentProperties,
              []
            );
            initialValuesJson = safeJsonParse<string[]>(initialValue, []);

            return {
              name,
              componentType,
              title: displayName,
              initialValues: initialValuesJson,
              value: initialValuesJson,
              ...componentPropertiesJson,
            };
          case INPUT_TYPES.KEY_VALUE_PAIR:
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
