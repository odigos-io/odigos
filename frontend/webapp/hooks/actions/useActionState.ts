import { ROUTES } from '@/utils';
import { useState } from 'react';
import { ActionItem } from '@/types';
import { useMutation } from 'react-query';
import { useActions } from './useActions';
import { useRouter } from 'next/navigation';
import { putAction, setAction, deleteAction } from '@/services';

interface Monitor {
  id: string;
  label: string;
  checked: boolean;
}

interface ActionState {
  id?: string;
  actionName: string;
  actionNote: string;
  actionData: any;
  selectedMonitors: Monitor[];
  disabled: boolean;
}

const DEFAULT_MONITORS: Monitor[] = [
  { id: '1', label: 'Logs', checked: true },
  { id: '2', label: 'Metrics', checked: true },
  { id: '3', label: 'Traces', checked: true },
];

export function useActionState() {
  const [actionState, setActionState] = useState<ActionState>({
    actionName: '',
    actionNote: '',
    actionData: null,
    selectedMonitors: DEFAULT_MONITORS,
    disabled: false,
  });

  const router = useRouter();
  const { getActionById } = useActions();

  const { mutateAsync: createAction } = useMutation((body: ActionItem) =>
    setAction(body)
  );
  const { mutateAsync: updateAction } = useMutation((body: ActionItem) =>
    putAction(actionState?.id, body)
  );

  const { mutateAsync: deleteActionMutation } = useMutation((id: string) =>
    deleteAction(id)
  );

  function onSuccess() {
    router.push(ROUTES.ACTIONS);
  }

  function onChangeActionState(key: string, value: any) {
    setActionState((prevState) => ({
      ...prevState,
      [key]: value,
    }));
    if (key === 'disabled') upsertAction(false);
  }

  function buildActionData(actionId: string) {
    const action = getActionById(actionId);

    const actionState = {
      id: action?.id,
      actionName: action?.spec?.actionName || '',
      actionNote: action?.spec?.notes || '',
      actionData: {
        clusterAttributes:
          action?.spec.clusterAttributes.map((attr, index) => ({
            attributeName: attr.attributeName,
            attributeStringValue: attr.attributeStringValue,
            id: index,
          })) || [],
      },
      selectedMonitors: DEFAULT_MONITORS.map((monitor) => ({
        ...monitor,

        checked: !!action?.spec?.signals.includes(monitor.label.toUpperCase()),
      })),
      disabled: action?.spec?.disabled || false,
    };

    setActionState(actionState);
  }

  async function upsertAction(callback: boolean = true) {
    const { actionName, actionNote, actionData, selectedMonitors, disabled } =
      actionState;

    const signals = selectedMonitors
      .filter((monitor) => monitor.checked)
      .map((monitor) => monitor.label.toUpperCase());

    const filteredActionData = filterEmptyActionDataFieldsByType(
      'add-cluster-info',
      actionData
    );

    const action = {
      actionName,
      notes: actionNote,
      signals,
      ...filteredActionData,
      disabled: !disabled,
    };

    try {
      if (actionState?.id) {
        await updateAction(action);
      } else {
        delete action.disabled;
        await createAction(action);
      }
      callback && onSuccess();
    } catch (error) {
      console.error({ error });
    }
  }

  function onDeleteAction() {
    try {
      if (actionState?.id) {
        deleteActionMutation(actionState.id);
        onSuccess();
      }
    } catch (error) {}
  }

  return {
    actionState,
    onChangeActionState,
    upsertAction,
    buildActionData,
    onDeleteAction,
  };
}

function filterEmptyActionDataFieldsByType(type: string, data: any) {
  switch (type) {
    case 'add-cluster-info':
      return {
        clusterAttributes: data.clusterAttributes.filter(
          (attr: any) =>
            attr.attributeStringValue !== '' && attr.attributeName !== ''
        ),
      };
    default:
      return data;
  }
}
