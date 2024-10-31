import { useState } from 'react';
import type { DrawerBaseItem } from '@/store';
import type { InstrumentationRuleInput, InstrumentationRuleSpec } from '@/types';

const INITIAL: InstrumentationRuleInput = {
  ruleName: '',
  notes: '',
  disabled: false,
  workloads: null,
  instrumentationLibraries: null,
  payloadCollection: {
    httpRequest: null,
    httpResponse: null,
    dbQuery: null,
    messaging: null,
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
        httpRequest: !!payloadCollection.httpRequest ? {} : null,
        httpResponse: !!payloadCollection.httpResponse ? {} : null,
        dbQuery: !!payloadCollection.dbQuery ? {} : null,
        messaging: !!payloadCollection.messaging ? {} : null,
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
