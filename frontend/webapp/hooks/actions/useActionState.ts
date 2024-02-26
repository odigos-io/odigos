import { useState } from 'react';

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

  function createNewAction() {
    const signals = selectedMonitors
      .filter((monitor) => monitor.checked)
      .map((monitor) => monitor.label.toUpperCase());

    const action = {
      actionName,
      notes: actionNote,
      signals,
      ...actionData,
    };

    console.log({ action });
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
