import { useGenericForm } from '@/hooks';
import { FORM_ALERTS } from '@/utils';
import { DrawerItem, useNotificationStore } from '@/store';
import { LatencySamplerSpec, type ActionDataParsed, type ActionInput } from '@/types';
import { ACTION_TYPE, isEmpty, NOTIFICATION_TYPE, safeJsonParse } from '@odigos/ui-components';

const INITIAL: ActionInput = {
  // @ts-ignore (TS complains about empty string because we expect an "ActionsType", but it's fine)
  type: '',
  name: '',
  notes: '',
  disable: false,
  signals: [],
  details: '',
};

type Errors = {
  type?: string;
  signals?: string;
  details?: string;
};

export function useActionFormData() {
  const { addNotification } = useNotificationStore();
  const { formData, formErrors, handleFormChange, handleErrorChange, resetFormData } = useGenericForm<ActionInput>(INITIAL);

  const validateForm = (params?: { withAlert?: boolean; alertTitle?: string }) => {
    const errors: Errors = {};
    let ok = true;

    Object.entries(formData).forEach(([k, v]) => {
      switch (k) {
        case 'type':
        case 'signals':
          if (isEmpty(v)) errors[k as keyof Errors] = FORM_ALERTS.FIELD_IS_REQUIRED;
          break;

        case 'details':
          if (isEmpty(v)) errors[k as keyof Errors] = FORM_ALERTS.FIELD_IS_REQUIRED;
          if (formData.type === ACTION_TYPE.LATENCY_SAMPLER) {
            (safeJsonParse(v as string, { endpoints_filters: [] }) as LatencySamplerSpec).endpoints_filters.forEach((endpoint) => {
              if (endpoint.http_route.charAt(0) !== '/') {
                errors[k as keyof Errors] = FORM_ALERTS.LATENCY_HTTP_ROUTE;
              }
            });
          }
          break;

        default:
          break;
      }
    });

    ok = !Object.values(errors).length;

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
