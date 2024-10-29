import { useState } from 'react';
import type { ActionDataParsed, ActionInput } from '@/types';
import { DrawerBaseItem } from '@/store';

const INITIAL: ActionInput = {
  type: '',
  name: '',
  notes: '',
  disable: false,
  signals: [],
  details: '',
};

export function useActionFormData() {
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
        case 'type':
        case 'signals':
        case 'details':
          if (Array.isArray(v) ? !v.length : !v) ok = false;
          break;

        default:
          break;
      }
    });

    return ok;
  };

  const loadFormWithDrawerItem = (drawerItem: DrawerBaseItem) => {
    const { type, spec } = drawerItem.item as ActionDataParsed;

    handleFormChange('type', type);

    Object.entries(spec).forEach(([k, v]) => {
      switch (k) {
        case 'actionName': {
          handleFormChange('name', v);
          break;
        }

        case 'disabled': {
          handleFormChange('disable', v);
          break;
        }

        case 'notes':
        case 'signals': {
          handleFormChange(k, v);
          break;
        }

        default: {
          handleFormChange('details', JSON.stringify({ [k]: v }));
          break;
        }
      }
    });
  };

  return {
    formData,
    handleFormChange,
    resetFormData,
    validateForm,
    loadFormWithDrawerItem,
  };
}
