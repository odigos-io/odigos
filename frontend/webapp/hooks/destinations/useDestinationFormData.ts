import { useState, useEffect } from 'react';
import { DrawerBaseItem } from '@/store';
import { useQuery } from '@apollo/client';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import { useConnectDestinationForm, useNotify } from '@/hooks';
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
  const { buildFormDynamicFields } = useConnectDestinationForm();

  const [formData, setFormData] = useState({ ...INITIAL });
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
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

  function handleFormChange(key: keyof typeof INITIAL | string, val: any) {
    // this is for a case where "exportedSignals" have been changed, it's an object so they children are targeted as: "exportedSignals.logs"
    const [parentKey, childKey] = key.split('.');

    if (!!childKey) {
      setFormData((prev) => ({
        ...prev,
        [parentKey]: {
          ...prev[parentKey],
          [childKey]: val,
        },
      }));
    } else {
      setFormData((prev) => ({
        ...prev,
        [parentKey]: val,
      }));
    }
  }

  const resetFormData = () => {
    setFormData({ ...INITIAL });
    setFormErrors({});
  };

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

    setFormErrors(errors);

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

    setFormData(updatedData);
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
