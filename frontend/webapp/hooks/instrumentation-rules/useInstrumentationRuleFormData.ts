import { FORM_ALERTS } from '@/utils';
import { useGenericForm } from '@/hooks';
import { useNotificationStore, type DrawerItem } from '@/store';
import { NOTIFICATION_TYPE, PayloadCollectionType, type InstrumentationRuleInput, type InstrumentationRuleSpec } from '@/types';

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
};

export function useInstrumentationRuleFormData() {
  const { addNotification } = useNotificationStore();
  const { formData, formErrors, handleFormChange, handleErrorChange, resetFormData } = useGenericForm<InstrumentationRuleInput>(INITIAL);

  const validateForm = (params?: { withAlert?: boolean; alertTitle?: string }) => {
    const errors: Partial<Record<keyof InstrumentationRuleInput, string>> = {};
    let ok = true;

    Object.entries(formData).forEach(([k, v]) => {
      switch (k) {
        case 'payloadCollection':
          const hasNoneSelected = !Object.values(v as InstrumentationRuleInput['payloadCollection']).filter((val) => !!val).length;
          if (hasNoneSelected) {
            ok = false;
            errors[k] = FORM_ALERTS.FIELD_IS_REQUIRED;
          }
          break;

        default:
          break;
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
    const { ruleName, notes, disabled, payloadCollection } = drawerItem.item as InstrumentationRuleSpec;

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
