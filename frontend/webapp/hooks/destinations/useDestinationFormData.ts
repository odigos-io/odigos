import { useState, useEffect } from 'react';
import { DrawerBaseItem } from '@/store';
import { useQuery } from '@apollo/client';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import { useConnectDestinationForm, useNotify } from '@/hooks';
import { ACTION, FORM_ALERTS, NOTIFICATION, safeJsonParse } from '@/utils';
import type { DynamicField, DestinationDetailsResponse, DestinationInput, DestinationTypeItem, ActualDestination, SupportedDestinationSignals } from '@/types';

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

  const [formData, setFormData] = useState({ ...INITIAL });
  const [dynamicFields, setDynamicFields] = useState<DynamicField[]>([]);

  const handleFormChange = (key: keyof typeof INITIAL | string, val: any) => {
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
  };

  const resetFormData = () => {
    setFormData({ ...INITIAL });
  };

  const validateForm = (params?: { withAlert?: boolean }) => {
    let ok = true;

    ok = dynamicFields.every((field) => (field.required ? !!field.value : true));

    if (!ok && params?.withAlert) {
      notify({
        type: NOTIFICATION.WARNING,
        title: ACTION.UPDATE,
        message: FORM_ALERTS.REQUIRED_FIELDS,
      });
    }

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

  const { buildFormDynamicFields } = useConnectDestinationForm();

  const t = destinationType || formData.type;
  const { data: { destinationTypeDetails } = {} } = useQuery<DestinationDetailsResponse>(GET_DESTINATION_TYPE_DETAILS, {
    variables: { type: t },
    skip: !t,
  });

  useEffect(() => {
    const { logs, metrics, traces } = supportedSignals || {};

    handleFormChange('exportedSignals', {
      logs: logs?.supported || false,
      metrics: metrics?.supported || false,
      traces: traces?.supported || false,
    });
  }, [supportedSignals]);

  useEffect(() => {
    if (destinationTypeDetails) {
      setDynamicFields(
        buildFormDynamicFields(destinationTypeDetails.fields).map((field) => {
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

  return {
    formData,
    handleFormChange,
    resetFormData,
    validateForm,
    loadFormWithDrawerItem,

    destinationTypeDetails,
    dynamicFields,
    setDynamicFields,
  };
}
