import { ROUTES } from '@/utils';
import { useState } from 'react';
import { useMutation } from 'react-query';
import { useActions } from './useActions';
import { useRouter } from 'next/navigation';
import { putAction, setAction, deleteAction } from '@/services';
import { ActionData, ActionItem, ActionState, ActionsType } from '@/types';

export interface Monitor {
  id: string;
  label: string;
  checked: boolean;
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
    type: '',
  });

  const router = useRouter();
  const { getActionById } = useActions();

  const { mutateAsync: createAction } = useMutation((body: ActionItem) =>
    setAction(body, actionState.type)
  );
  const { mutateAsync: updateAction } = useMutation((body: ActionItem) =>
    putAction(actionState?.id, body, actionState.type)
  );

  const { mutateAsync: deleteActionMutation } = useMutation((id: string) =>
    deleteAction(id, actionState.type)
  );

  async function onSuccess() {
    router.push(ROUTES.ACTIONS);
  }

  function onChangeActionState(key: string, value: any) {
    setActionState((prevState) => ({
      ...prevState,
      [key]: value,
    }));
    if (key === 'disabled') upsertAction(false);
  }

  async function buildActionData(actionId: string) {
    const action = await getActionById(actionId);

    const actionState = {
      id: action?.id,
      actionName: action?.spec?.actionName || '',
      actionNote: action?.spec?.notes || '',
      type: action?.type || '',
      actionData: getActionDataByType(action),
      selectedMonitors: DEFAULT_MONITORS.map((monitor) => ({
        ...monitor,

        checked: !!action?.spec?.signals.includes(monitor.label.toUpperCase()),
      })),
      disabled: action?.spec?.disabled || false,
    };

    setActionState(actionState);
  }

  async function upsertAction(callback: boolean = true) {
    const {
      actionName,
      actionNote,
      actionData,
      selectedMonitors,
      disabled,
      type,
    } = actionState;

    const signals = getSupportedSignals(type, selectedMonitors)
      .filter((monitor) => monitor.checked)
      .map((monitor) => monitor.label.toUpperCase());

    const filteredActionData = filterEmptyActionDataFieldsByType(
      type,
      actionData
    );

    const action = {
      actionName,
      notes: actionNote,
      signals,
      ...filteredActionData,
      disabled: callback ? disabled : !disabled,
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

  function getSupportedSignals(type: string, signals: Monitor[]) {
    if (
      type === ActionsType.ERROR_SAMPLER ||
      type === ActionsType.PROBABILISTIC_SAMPLER ||
      type === ActionsType.LATENCY_SAMPLER ||
      type === ActionsType.PII_MASKING
    ) {
      return signals.filter((signal) => signal.label === 'Traces');
    }

    return signals;
  }

  return {
    actionState,
    upsertAction,
    onDeleteAction,
    buildActionData,
    getSupportedSignals,
    onChangeActionState,
  };
}

function filterEmptyActionDataFieldsByType(type: string, data: any) {
  switch (type) {
    case ActionsType.ADD_CLUSTER_INFO:
      return {
        clusterAttributes: data.clusterAttributes.filter(
          (attr: any) =>
            attr.attributeStringValue !== '' && attr.attributeName !== ''
        ),
      };
    case ActionsType.DELETE_ATTRIBUTES:
      return {
        attributeNamesToDelete: data.attributeNamesToDelete.filter(
          (attr: string) => attr !== ''
        ),
      };
    case ActionsType.RENAME_ATTRIBUTES:
      return {
        renames: Object.fromEntries(
          Object.entries(data.renames).filter(
            ([key, value]: [string, string]) => key !== '' && value !== ''
          )
        ),
      };
    default:
      return data;
  }
}

function getActionDataByType(action: ActionData | undefined) {
  if (!action) return {};
  switch (action.type) {
    case ActionsType.ADD_CLUSTER_INFO:
      return {
        clusterAttributes: action.spec.clusterAttributes.map((attr, index) => ({
          attributeName: attr.attributeName,
          attributeStringValue: attr.attributeStringValue,
          id: index,
        })),
      };
    case ActionsType.DELETE_ATTRIBUTES:
      return {
        attributeNamesToDelete: action.spec.attributeNamesToDelete,
      };
    case ActionsType.RENAME_ATTRIBUTES:
      return {
        renames: action.spec.renames,
      };
    case ActionsType.ERROR_SAMPLER:
      return {
        fallback_sampling_ratio: action.spec.fallback_sampling_ratio,
      };
    case ActionsType.PROBABILISTIC_SAMPLER:
      return {
        sampling_percentage: action.spec.sampling_percentage,
      };
    case ActionsType.LATENCY_SAMPLER:
      return {
        endpoints_filters: action.spec.endpoints_filters,
      };
    default:
      return {};
  }
}
