import { setAction } from '@/services';
import { ActionItem } from '@/types';
import { ROUTES } from '@/utils';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { useMutation } from 'react-query';

const DEFAULT_MONITORS = [
  { id: '1', label: 'Logs', checked: true },
  { id: '2', label: 'Metrics', checked: true },
  { id: '3', label: 'Traces', checked: true },
];

export function useActionState() {
  const [actionName, setActionName] = useState<string>('');
  const [actionNote, setActionNote] = useState<string>('');
  const [actionData, setActionData] = useState<any>(null);
  const [selectedMonitors, setSelectedMonitors] =
    useState<{ id: string; label: string; checked: boolean }[]>(
      DEFAULT_MONITORS
    );

  const router = useRouter();
  const { mutateAsync } = useMutation((body: ActionItem) => setAction(body));

  function onCreateSuccess() {
    router.push(ROUTES.ACTIONS);
  }

  async function createNewAction() {
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
    actionName,
    setActionName,
    actionNote,
    setActionNote,
    selectedMonitors,
    setSelectedMonitors,
    actionData,
    setActionData,
    createNewAction,
  };
}
