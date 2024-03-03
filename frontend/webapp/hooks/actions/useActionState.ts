import { ROUTES } from '@/utils';
import { useState } from 'react';
import { ActionItem } from '@/types';
import { putAction, setAction } from '@/services';
import { useMutation } from 'react-query';
import { useRouter } from 'next/navigation';
import { capitalizeFirstLetter } from '@/utils/functions';
import { useActions } from './useActions';

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
  });

  const router = useRouter();
  const { getActionById } = useActions();

  const { mutateAsync } = useMutation((body: ActionItem) => setAction(body));
  const { mutateAsync: updateAction } = useMutation((body: ActionItem) =>
    putAction(actionState?.id, body)
  );
  function onCreateSuccess() {
    router.push(ROUTES.ACTIONS);
  }

  function onChangeActionState(key: string, value: any) {
    setActionState((prevState) => ({
      ...prevState,
      [key]: value,
    }));
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
    };

    setActionState(actionState);
  }

  async function createNewAction() {
    const { actionName, actionNote, actionData, selectedMonitors } =
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
    };

    try {
      await mutateAsync(action);
      onCreateSuccess();
    } catch (error) {
      console.error({ error });
    }
  }

  async function updateCurrentAction() {
    const { actionName, actionNote, actionData, selectedMonitors } =
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
    };

    console.log({ action });

    try {
      await updateAction(action);
      onCreateSuccess();
    } catch (error) {
      console.error({ error });
    }
  }

  return {
    actionState,
    onChangeActionState,
    createNewAction,
    updateCurrentAction,
    buildActionData,
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
