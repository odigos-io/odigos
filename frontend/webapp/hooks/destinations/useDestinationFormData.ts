import { useState, useEffect } from 'react';
import { DrawerBaseItem } from '@/store';
import { useQuery } from '@apollo/client';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import { useConnectDestinationForm, useGenericForm, useNotify } from '@/hooks';
import { ACTION, FORM_ALERTS, NOTIFICATION, safeJsonParse } from '@/utils';
import {
  type DynamicField,
  type DestinationDetailsResponse,
  type DestinationInput,
  type DestinationTypeItem,
  type ActualDestination,
  type SupportedDestinationSignals,
  OVERVIEW_ENTITY_TYPES,
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

export function useDestinationFormData(params?: { destinationType?: string; supportedSignals?: SupportedDestinationSignals; preLoadedFields?: string | DestinationTypeItem['fields'] }) {
  const { destinationType, supportedSignals, preLoadedFields } = params || {};

  const notify = useNotify();
  const { formData, formErrors, handleFormChange, handleErrorChange, resetFormData } = useGenericForm<DestinationInput>(INITIAL);

  const { buildFormDynamicFields } = useConnectDestinationForm();
  const [dynamicFields, setDynamicFields] = useState<DynamicField[]>([]);

  const t = destinationType || formData.type;
  const { data: { destinationTypeDetails } = {} } = useQuery<DestinationDetailsResponse>(GET_DESTINATION_TYPE_DETAILS, {
    variables: { type: t },
    skip: !t,
    onError: (error) => notify({ type: NOTIFICATION.ERROR, title: ACTION.FETCH, message: error.message, crdType: OVERVIEW_ENTITY_TYPES.DESTINATION }),
  });

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
    const errors = {};
    let ok = true;

    dynamicFields.forEach(({ name, value, required }) => {
      if (required && !value) {
        ok = false;
        errors[name] = FORM_ALERTS.FIELD_IS_REQUIRED;
      }
    });

    if (!ok && params?.withAlert) {
      notify({
        type: NOTIFICATION.WARNING,
        title: params.alertTitle,
        message: FORM_ALERTS.REQUIRED_FIELDS,
      });
    }

    handleErrorChange(undefined, undefined, errors);

    return ok;
  };

  const loadFormWithDrawerItem = (drawerItem: DrawerBaseItem) => {
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
