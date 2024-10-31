import { useState } from 'react';
// import { DrawerBaseItem } from '@/store';
import type { InstrumentationRuleInput } from '@/types';

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

  // const loadFormWithDrawerItem = (drawerItem: DrawerBaseItem) => {
  //   const { type, spec } = drawerItem.item as ActionDataParsed;

  //   const updatedData: ActionInput = {
  //     ...INITIAL,
  //     type,
  //   };

  //   Object.entries(spec).forEach(([k, v]) => {
  //     switch (k) {
  //       case 'actionName': {
  //         updatedData['name'] = v;
  //         break;
  //       }

  //       case 'disabled': {
  //         updatedData['disable'] = v;
  //         break;
  //       }

  //       case 'notes':
  //       case 'signals': {
  //         updatedData[k] = v;
  //         break;
  //       }

  //       default: {
  //         updatedData['details'] = JSON.stringify({ [k]: v });
  //         break;
  //       }
  //     }
  //   });

  //   setFormData(updatedData);
  // };

  return {
    formData,
    handleFormChange,
    resetFormData,
    validateForm,
    // loadFormWithDrawerItem,
  };
}
