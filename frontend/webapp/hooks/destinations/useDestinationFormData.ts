import { useState, useEffect } from 'react';
import { useGenericForm } from '@/hooks';
import { useQuery } from '@apollo/client';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import { type DrawerItem, useNotificationStore } from '@/store';
import { ACTION, FORM_ALERTS, INPUT_TYPES, safeJsonParse } from '@/utils';
import {
  type DynamicField,
  type DestinationDetailsResponse,
  type DestinationInput,
  type DestinationTypeItem,
  type ActualDestination,
  type SupportedDestinationSignals,
  OVERVIEW_ENTITY_TYPES,
  NOTIFICATION_TYPE,
  type DestinationDetailsField,
} from '@/types';

const INITIAL: DestinationInput = {
  type: '',
  name: '',
  exportedSignals: {
    logs: false,
    metrics: false,
    traces: false,
  },
  fields: [],
};

const buildFormDynamicFields = (fields: DestinationDetailsField[]): DynamicField[] => {
  return fields
    .map((field) => {
      const { name, componentType, componentProperties, displayName, initialValue, renderCondition } = field;

      switch (componentType) {
        case INPUT_TYPES.DROPDOWN: {
          const componentPropertiesJson = safeJsonParse<{ [key: string]: string }>(componentProperties, {});
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
            name,
            componentType,
            title: displayName,
            value: initialValue,
            placeholder: componentPropertiesJson.placeholder || 'Select an option',
            options,
            renderCondition,
            ...componentPropertiesJson,
          };
        }

        default: {
          const componentPropertiesJson = safeJsonParse<{ [key: string]: string }>(componentProperties, {});

          return {
            name,
            componentType,
            title: displayName,
            value: initialValue,
            renderCondition,
            ...componentPropertiesJson,
          };
        }
      }
    })
    .filter((field): field is DynamicField => field !== undefined);
};

const destinationTypeDetails = {
  fields: [
    {
      name: 'CLICKHOUSE_ENDPOINT',
      displayName: 'Endpoint',
      componentType: 'input',
      componentProperties:
        '{"placeholder":"http://host:port","required":true,"tooltip":"The ClickHouse endpoint is the URL where the ClickHouse server is listening for incoming connections.","type":"text"}',
      secret: false,
      initialValue: '',
      renderCondition: [],
      hideFromReadData: [],
      customReadDataLabels: [],
    },
    {
      name: 'CLICKHOUSE_USERNAME',
      displayName: 'Username',
      componentType: 'input',
      componentProperties: '{"required":false,"tooltip":"If Clickhouse Authentication is used, provide the username","type":"text"}',
      secret: false,
      initialValue: '',
      renderCondition: [],
      hideFromReadData: [],
      customReadDataLabels: [],
    },
    {
      name: 'CLICKHOUSE_PASSWORD',
      displayName: 'Password',
      componentType: 'input',
      componentProperties: '{"required":false,"tooltip":"If Clickhouse Authentication is used, provide the password","type":"password"}',
      secret: true,
      initialValue: '',
      renderCondition: [],
      hideFromReadData: [],
      customReadDataLabels: [],
    },
    {
      name: 'CLICKHOUSE_CREATE_SCHEME',
      displayName: 'Create Scheme',
      componentType: 'checkbox',
      componentProperties:
        '{"required":true,"tooltip":"Should the destination create the schema for you? Set to `false` if you manage your own schema, or `true` to have Odigos create the schema for you"}',
      secret: false,
      initialValue: 'true',
      renderCondition: [],
      hideFromReadData: [],
      customReadDataLabels: [],
    },
    {
      name: 'CLICKHOUSE_DATABASE_NAME',
      displayName: 'Database Name',
      componentType: 'input',
      componentProperties:
        '{"required":true,"tooltip":"The name of the Clickhouse Database where the telemetry data will be stored. The Database will not be created when not exists, so make sure you have created it before","type":"text"}',
      secret: false,
      initialValue: 'otel',
      renderCondition: [],
      hideFromReadData: [],
      customReadDataLabels: [],
    },
    {
      name: 'CLICKHOUSE_TRACES_TABLE',
      displayName: 'Traces Table',
      componentType: 'input',
      componentProperties: '{"required":true,"tooltip":"Name of the ClickHouse Table to use for storing trace spans. This name should be used in span queries","type":"text"}',
      secret: false,
      initialValue: 'otel_traces',
      renderCondition: [],
      hideFromReadData: [],
      customReadDataLabels: [],
    },
    {
      name: 'CLICKHOUSE_METRICS_TABLE',
      displayName: 'Metrics Table',
      componentType: 'input',
      componentProperties: '{"required":true,"tooltip":"Name of the ClickHouse Table to use for storing metrics. This name should be used in metric queries","type":"text"}',
      secret: false,
      initialValue: 'otel_metrics',
      renderCondition: [],
      hideFromReadData: [],
      customReadDataLabels: [],
    },
    {
      name: 'CLICKHOUSE_LOGS_TABLE',
      displayName: 'Logs Table',
      componentType: 'input',
      componentProperties: '{"required":true,"tooltip":"Name of the ClickHouse Table to use for storing logs. This name should be used in log queries","type":"text"}',
      secret: false,
      initialValue: 'otel_logs',
      renderCondition: [],
      hideFromReadData: [],
      customReadDataLabels: [],
    },
  ],
};

export function useDestinationFormData(params?: { destinationType?: string; supportedSignals?: SupportedDestinationSignals; preLoadedFields?: string | DestinationTypeItem['fields'] }) {
  const { destinationType, supportedSignals, preLoadedFields } = params || {};

  const { addNotification } = useNotificationStore();
  const { formData, formErrors, handleFormChange, handleErrorChange, resetFormData } = useGenericForm<DestinationInput>(INITIAL);

  const [dynamicFields, setDynamicFields] = useState<DynamicField[]>([]);

  const t = destinationType || formData.type;
  // const { data: { destinationTypeDetails } = {} } = useQuery<DestinationDetailsResponse>(GET_DESTINATION_TYPE_DETAILS, {
  //   variables: { type: t },
  //   skip: !t,
  //   onError: (error) =>
  //     addNotification({
  //       type: NOTIFICATION_TYPE.ERROR,
  //       title: error.name || ACTION.FETCH,
  //       message: error.cause?.message || error.message,
  //       crdType: OVERVIEW_ENTITY_TYPES.DESTINATION,
  //     }),
  // });

  useEffect(() => {
    if (destinationTypeDetails) {
      setDynamicFields(
        buildFormDynamicFields(destinationTypeDetails.fields).map((field) => {
          // if we have preloaded fields, we need to set the value of the field
          // (this can be from an odigos-detected-destination during create, or from an existing destination during edit/update)
          if (!!preLoadedFields) {
            const parsedFields = typeof preLoadedFields === 'string' ? safeJsonParse<Record<string, string>>(preLoadedFields, {}) : preLoadedFields;

            if (field.name in parsedFields) {
              return {
                ...field,
                value: parsedFields[field.name],
              };
            }
          }

          return field;
        }),
      );
    } else {
      setDynamicFields([]);
    }
  }, [destinationTypeDetails, preLoadedFields]);

  useEffect(() => {
    handleFormChange(
      'fields',
      dynamicFields.map((field) => ({
        key: field.name,
        value: field.value,
      })),
    );
  }, [dynamicFields]);

  useEffect(() => {
    const { logs, metrics, traces } = supportedSignals || {};

    handleFormChange('exportedSignals', {
      logs: logs?.supported || false,
      metrics: metrics?.supported || false,
      traces: traces?.supported || false,
    });
  }, [supportedSignals]);

  const validateForm = (params?: { withAlert?: boolean; alertTitle?: string }) => {
    const errors: Record<DynamicField['name'], string> = {};
    let ok = true;

    dynamicFields.forEach(({ name, value, required }) => {
      if (required && !value) {
        ok = false;
        errors[name] = FORM_ALERTS.FIELD_IS_REQUIRED;
      }
    });

    if (!ok && params?.withAlert) {
      addNotification({
        type: NOTIFICATION_TYPE.WARNING,
        title: params.alertTitle,
        message: FORM_ALERTS.REQUIRED_FIELDS,
        hideFromHistory: true,
      });
    }

    handleErrorChange(undefined, undefined, errors);

    return ok;
  };

  const loadFormWithDrawerItem = (drawerItem: DrawerItem) => {
    const {
      destinationType: { type },
      name,
      exportedSignals,
      fields,
    } = drawerItem.item as ActualDestination;

    const updatedData: DestinationInput = {
      ...INITIAL,
      type,
      name,
      exportedSignals,
      fields: Object.entries(safeJsonParse(fields, {})).map(([key, value]: [string, string]) => ({ key, value })),
    };

    handleFormChange(undefined, undefined, updatedData);
  };

  return {
    formData,
    formErrors,
    handleFormChange,
    resetFormData,
    validateForm,
    loadFormWithDrawerItem,

    destinationTypeDetails,
    dynamicFields,
    setDynamicFields,
  };
}
