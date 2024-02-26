import { ROUTES } from '@/utils';
import { useState } from 'react';
import { ActionItem } from '@/types';
import { setAction } from '@/services';
import { useMutation } from 'react-query';
import { useRouter } from 'next/navigation';

interface Monitor {
  id: string;
  label: string;
  checked: boolean;
}

interface ActionState {
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
  const { mutateAsync } = useMutation((body: ActionItem) => setAction(body));

  function onCreateSuccess() {
    router.push(ROUTES.ACTIONS);
  }

  function onChangeActionState(key: string, value: any) {
    setActionState((prevState) => ({
      ...prevState,
      [key]: value,
    }));
  }

  async function createNewAction() {
    const { actionName, actionNote, actionData, selectedMonitors } =
      actionState;

    const signals = selectedMonitors
      .filter((monitor) => monitor.checked)
      .map((monitor) => monitor.label.toUpperCase());

    const action = {
      actionName,
      notes: actionNote,
      signals,
      ...actionData,
    };

    try {
      await mutateAsync(action);
      onCreateSuccess();
    } catch (error) {
      console.error({ error });
    }
  }

  return {
    actionState,
    onChangeActionState,
    createNewAction,
  };
}
