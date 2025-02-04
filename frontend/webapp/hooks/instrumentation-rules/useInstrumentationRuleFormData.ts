import { FORM_ALERTS } from '@/utils';
import { useGenericForm } from '@/hooks';
import { NOTIFICATION_TYPE } from '@odigos/ui-utils';
import { type DrawerItem, useNotificationStore } from '@odigos/ui-containers';
import { CodeAttributesType, PayloadCollectionType, type InstrumentationRuleInput, type InstrumentationRuleSpec } from '@/types';

const INITIAL: InstrumentationRuleInput = {
  ruleName: '',
  notes: '',
  disabled: false,
  workloads: null,
  instrumentationLibraries: null,
  payloadCollection: {
    [PayloadCollectionType.HTTP_REQUEST]: null,
    [PayloadCollectionType.HTTP_RESPONSE]: null,
    [PayloadCollectionType.DB_QUERY]: null,
    [PayloadCollectionType.MESSAGING]: null,
  },
  codeAttributes: {
    [CodeAttributesType.COLUMN]: null,
    [CodeAttributesType.FILE_PATH]: null,
    [CodeAttributesType.FUNCTION]: null,
    [CodeAttributesType.LINE_NUMBER]: null,
    [CodeAttributesType.NAMESPACE]: null,
    [CodeAttributesType.STACKTRACE]: null,
  },
};

export function useInstrumentationRuleFormData() {
  const { addNotification } = useNotificationStore();
  const { formData, formErrors, handleFormChange, handleErrorChange, resetFormData } = useGenericForm<InstrumentationRuleInput>(INITIAL);

  const validateForm = (params?: { withAlert?: boolean; alertTitle?: string }) => {
    const errors: Partial<Record<keyof InstrumentationRuleInput, string>> = {};
    let ok = true;

    // Instru Rules don't have any specific validations yet, no required fields at this time

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
    const { ruleName, notes, disabled, payloadCollection, codeAttributes } = drawerItem.item as InstrumentationRuleSpec;

    const updatedData: InstrumentationRuleInput = {
      ...INITIAL,
      ruleName,
      notes,
      disabled,
    };

    if (payloadCollection) {
      updatedData['payloadCollection'] = {
        [PayloadCollectionType.HTTP_REQUEST]: !!payloadCollection[PayloadCollectionType.HTTP_REQUEST] ? {} : null,
        [PayloadCollectionType.HTTP_RESPONSE]: !!payloadCollection[PayloadCollectionType.HTTP_RESPONSE] ? {} : null,
        [PayloadCollectionType.DB_QUERY]: !!payloadCollection[PayloadCollectionType.DB_QUERY] ? {} : null,
        [PayloadCollectionType.MESSAGING]: !!payloadCollection[PayloadCollectionType.MESSAGING] ? {} : null,
      };
    }

    if (codeAttributes) {
      updatedData['codeAttributes'] = {
        [CodeAttributesType.COLUMN]: codeAttributes[CodeAttributesType.COLUMN] || null,
        [CodeAttributesType.FILE_PATH]: codeAttributes[CodeAttributesType.FILE_PATH] || null,
        [CodeAttributesType.FUNCTION]: codeAttributes[CodeAttributesType.FUNCTION] || null,
        [CodeAttributesType.LINE_NUMBER]: codeAttributes[CodeAttributesType.LINE_NUMBER] || null,
        [CodeAttributesType.NAMESPACE]: codeAttributes[CodeAttributesType.NAMESPACE] || null,
        [CodeAttributesType.STACKTRACE]: codeAttributes[CodeAttributesType.STACKTRACE] || null,
      };
    }

    handleFormChange(undefined, undefined, updatedData);
  };

  return {
    formData,
    formErrors,
    handleFormChange,
    resetFormData,
    validateForm,
    loadFormWithDrawerItem,
  };
}
