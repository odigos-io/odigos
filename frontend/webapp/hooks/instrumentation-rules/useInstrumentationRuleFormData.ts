import { useState } from 'react';
import type { DrawerBaseItem } from '@/store';
import { PayloadCollectionType, type InstrumentationRuleInput, type InstrumentationRuleSpec } from '@/types';

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
  const [formData, setFormData] = useState({ ...INITIAL });

  const handleFormChange = (key: keyof typeof INITIAL, val: any) => {
    setFormData((prev) => ({
      ...prev,
      [key]: val,
    }));
  };

  const resetFormData = () => {
    setFormData({ ...INITIAL });
  };

  const validateForm = () => {
    let ok = true;

    Object.entries(formData).forEach(([k, v]) => {
      switch (k) {
        case 'payloadCollection':
          const hasNoneSelected = !Object.values(v as InstrumentationRuleInput['payloadCollection']).filter((val) => !!val).length;
          ok = !hasNoneSelected;
          break;

        default:
          break;
      }
    });

    return ok;
  };

  const loadFormWithDrawerItem = (drawerItem: DrawerBaseItem) => {
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

    setFormData(updatedData);
  };

  return {
    formData,
    handleFormChange,
    resetFormData,
    validateForm,
    loadFormWithDrawerItem,
  };
}
