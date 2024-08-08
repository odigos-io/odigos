import { safeJsonParse } from '@/utils';
import {
  DestinationDetailsField,
  DestinationInput,
  DynamicField,
} from '@/types';
import { CREATE_DESTINATION } from '@/graphql';
import { useMutation } from '@apollo/client';

export function useConnectDestinationForm() {
  const [createDestination] = useMutation(CREATE_DESTINATION);
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

          //   case 'multi_input':
          //   case 'textarea':
          //     return {
          //       name,
          //       componentType,
          //       title: displayName,
          //       ...componentPropertiesJson,
          //     };
          default:
            return undefined;
        }
      })
      .filter((field): field is DynamicField => field !== undefined);
  }

  async function createNewDestination(destination: DestinationInput) {
    try {
      const { data } = await createDestination({
        variables: { destination },
      });
      return data?.createNewDestination?.id;
    } catch (error) {
      console.error('Error creating new destination:', error);
      throw error;
    }
  }

  return { buildFormDynamicFields, createNewDestination };
}
