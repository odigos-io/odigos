import { useState } from 'react';
import { useNotify } from '../notification/useNotify';
import { DrawerBaseItem } from '@/store';
import { ACTION, FORM_ALERTS, NOTIFICATION } from '@/utils';
import type { ActionDataParsed, ActionInput } from '@/types';

const INITIAL: ActionInput = {
  type: '',
  name: '',
  notes: '',
  disable: false,
  signals: [],
  details: '',
};

export function useActionFormData() {
  const notify = useNotify();
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

  const validateForm = (params?: { withAlert?: boolean }) => {
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

    if (!ok && params?.withAlert) {
      notify({
        type: NOTIFICATION.ERROR,
        title: ACTION.UPDATE,
        message: FORM_ALERTS.REQUIRED_FIELDS,
      });
    }

    return ok;
  };

  const loadFormWithDrawerItem = (drawerItem: DrawerBaseItem) => {
    const { type, spec } = drawerItem.item as ActionDataParsed;

    const updatedData: ActionInput = {
      ...INITIAL,
      type,
    };

    Object.entries(spec).forEach(([k, v]) => {
      switch (k) {
        case 'actionName': {
          updatedData['name'] = v;
          break;
        }

        case 'disabled': {
          updatedData['disable'] = v;
          break;
        }

        case 'notes':
        case 'signals': {
          updatedData[k] = v;
          break;
        }

        default: {
          updatedData['details'] = JSON.stringify({ [k]: v });
          break;
        }
      }
    });

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
