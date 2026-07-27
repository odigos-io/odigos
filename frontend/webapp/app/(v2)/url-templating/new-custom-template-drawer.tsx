'use client';

import React, { useEffect, useMemo, useState } from 'react';
import styled from 'styled-components';
import { ActionIcon, PlusIcon } from '@odigos/ui-kit/icons';
import type { Action } from '@odigos/ui-kit/types';
import { ActionType, SignalType, StatusType } from '@odigos/ui-kit/types';
import {
  Badge,
  Button,
  ButtonSize,
  ButtonVariants,
  Drawer,
  DropDown,
  FlexColumn,
  FlexRow,
  Input,
  Note,
  RadioCard,
  Search,
  Typography,
  TypographyColor,
  TypographySize,
} from '@odigos/ui-kit/components';
import { useApiMutation } from '@odigos/ui-kit/contexts';
import { ActionOriginBadge, actionOriginLabel, isActionUiGenerated } from './action-origin-badge';
import { ScopeTokens } from './scope-tokens';
import type { CustomTemplateCrGroup } from './url-templatization-aggregate';
import {
  UI_TEMPLATE_RULES_ACTION_NAME,
  customTemplateGroupSearchText,
  findEntireClusterGroupForAction,
  findUiTemplateRulesAction,
} from './ui-template-rules';

const DrawerBody = styled(FlexColumn)`
  width: 100%;
  padding: 0 24px 8px;
  gap: 16px;
`;

const DrawerSaveFooter = styled(FlexColumn)`
  width: 100%;
  gap: 12px;
  padding: 16px 24px;
  box-sizing: border-box;
  border-top: 1px solid ${({ theme }) => theme.v2.colors.silver[700]};
  background: ${({ theme }) => theme.v2.colors.black[500]};
`;

const DrawerSaveFooterActions = styled(FlexRow)`
  width: 100%;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
`;

const OptionsColumn = styled(FlexColumn)`
  width: 100%;
  gap: 10px;
`;

const DetailPanel = styled(FlexColumn)`
  width: 100%;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 8px;
  background: ${({ theme }) => theme.v2.colors.silver[900]};
  border: 1px solid ${({ theme }) => theme.v2.colors.silver[700]};
`;

const GroupOption = styled.button<{ $selected?: boolean }>`
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  text-align: left;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  border: 1px solid
    ${({ theme, $selected }) => ($selected ? theme.v2.colors.blue[400] : theme.v2.colors.silver[700])};
  background: ${({ theme, $selected }) =>
    $selected ? theme.v2.colors.silver[800] : theme.v2.colors.silver[900]};
  color: inherit;
  font: inherit;

  &:hover {
    background: ${({ theme }) => theme.v2.colors.silver[800]};
  }

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.v2.colors.blue[400]};
    outline-offset: 2px;
  }
`;

const GroupList = styled(FlexColumn)`
  width: 100%;
  gap: 8px;
  max-height: 320px;
  overflow-y: auto;
`;

const UI_TEMPLATE_RULES_OPTION_ID = '__ui_template_rules__';
const NEW_ACTION_OPTION_ID = '__new_action__';

export type NewCustomTemplatePath = 'existing' | 'new_group' | 'entire_cluster';

export type NewCustomTemplateTarget =
  | { kind: 'edit_group'; group: CustomTemplateCrGroup }
  | { kind: 'create_group'; action: Action; lockEntireClusterScope?: boolean };

type NewCustomTemplateDrawerProps = {
  isOpen: boolean;
  actions: Action[];
  groups: CustomTemplateCrGroup[];
  onClose: () => void;
  onContinue: (target: NewCustomTemplateTarget) => void;
};

export function NewCustomTemplateDrawer({
  isOpen,
  actions,
  groups,
  onClose,
  onContinue,
}: NewCustomTemplateDrawerProps) {
  const [path, setPath] = useState<NewCustomTemplatePath>('existing');
  const [groupQuery, setGroupQuery] = useState('');
  const [selectedGroupKey, setSelectedGroupKey] = useState<string | null>(null);
  const [selectedActionId, setSelectedActionId] = useState(UI_TEMPLATE_RULES_OPTION_ID);
  const [newActionName, setNewActionName] = useState('');
  const [error, setError] = useState<string | null>(null);

  const [createAction, { loading: creatingAction }] = useApiMutation('CREATE_ACTION', {
    refetchQueries: ['GetActions'],
    awaitRefetchQueries: true,
  });

  useEffect(() => {
    if (!isOpen) {
      return;
    }
    setPath('existing');
    setGroupQuery('');
    setSelectedGroupKey(null);
    setNewActionName('');
    setError(null);
    const uiAction = findUiTemplateRulesAction(actions);
    setSelectedActionId(uiAction?.id ?? UI_TEMPLATE_RULES_OPTION_ID);
  }, [isOpen, actions]);

  const filteredGroups = useMemo(() => {
    const query = groupQuery.trim().toLowerCase();
    const sorted = [...groups].sort((a, b) => {
      const actionCmp = a.actionLabel.localeCompare(b.actionLabel);
      if (actionCmp !== 0) {
        return actionCmp;
      }
      return a.groupIndex - b.groupIndex;
    });
    if (!query) {
      return sorted;
    }
    return sorted.filter((group) => customTemplateGroupSearchText(group).includes(query));
  }, [groupQuery, groups]);

  const selectedGroup = useMemo(() => {
    if (!selectedGroupKey) {
      return null;
    }
    return groups.find((group) => groupKey(group) === selectedGroupKey) ?? null;
  }, [groups, selectedGroupKey]);

  const creatingNewAction = selectedActionId === NEW_ACTION_OPTION_ID;

  const actionOptions = useMemo(() => {
    const uiAction = findUiTemplateRulesAction(actions);
    const options = actions.map((action) => ({
      id: action.id,
      label: action.name?.trim() || action.id,
      badge: action.disabled ? 'Disabled' : actionOriginLabel(isActionUiGenerated(action)),
    }));
    if (!uiAction) {
      options.unshift({
        id: UI_TEMPLATE_RULES_OPTION_ID,
        label: UI_TEMPLATE_RULES_ACTION_NAME,
        badge: 'Create',
      });
    }
    options.push({
      id: NEW_ACTION_OPTION_ID,
      label: 'Create new action…',
      badge: 'New',
    });
    return options;
  }, [actions]);

  const createUrlTemplatizationAction = async (name: string, notes?: string): Promise<Action> => {
    const trimmed = name.trim();
    if (!trimmed) {
      throw new Error('Enter a name for the new action.');
    }
    const { data, error: createError } = await createAction({
      action: {
        type: ActionType.URLTemplatization,
        name: trimmed,
        notes: notes ?? null,
        disabled: false,
        signals: [SignalType.Traces],
        fields: {
          urlTemplatizationRulesGroups: [],
        },
      },
    });
    if (createError) {
      throw new Error(createError.message || `Failed to create action "${trimmed}"`);
    }
    const created = data?.createAction;
    if (!created) {
      throw new Error(`Failed to create action "${trimmed}"`);
    }
    return created;
  };

  const ensureUiTemplateRulesAction = async (): Promise<Action> => {
    const existing = findUiTemplateRulesAction(actions);
    if (existing) {
      return existing;
    }
    return createUrlTemplatizationAction(
      UI_TEMPLATE_RULES_ACTION_NAME,
      'Default action for URL templates created from the Odigos UI.',
    );
  };

  const resolveSelectedAction = async (): Promise<Action> => {
    if (selectedActionId === UI_TEMPLATE_RULES_OPTION_ID) {
      return ensureUiTemplateRulesAction();
    }
    if (selectedActionId === NEW_ACTION_OPTION_ID) {
      return createUrlTemplatizationAction(newActionName);
    }
    const action = actions.find((candidate) => candidate.id === selectedActionId);
    if (!action) {
      throw new Error('Selected action was not found.');
    }
    return action;
  };

  const canContinue = useMemo(() => {
    if (creatingAction) {
      return false;
    }
    if (path === 'existing') {
      return !!selectedGroup;
    }
    if (path === 'new_group') {
      if (creatingNewAction) {
        return !!newActionName.trim();
      }
      return !!selectedActionId;
    }
    return true;
  }, [creatingAction, creatingNewAction, newActionName, path, selectedActionId, selectedGroup]);

  const handleContinue = async () => {
    if (!canContinue) {
      return;
    }
    setError(null);
    try {
      if (path === 'existing') {
        if (!selectedGroup) {
          return;
        }
        onContinue({ kind: 'edit_group', group: selectedGroup });
        onClose();
        return;
      }

      if (path === 'new_group') {
        const action = await resolveSelectedAction();
        onContinue({ kind: 'create_group', action });
        onClose();
        return;
      }

      const action = await ensureUiTemplateRulesAction();
      const existingClusterGroup = findEntireClusterGroupForAction(groups, action.id);
      if (existingClusterGroup) {
        onContinue({ kind: 'edit_group', group: existingClusterGroup });
      } else {
        onContinue({ kind: 'create_group', action, lockEntireClusterScope: true });
      }
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to continue');
    }
  };

  return (
    <Drawer
      isOpen={isOpen}
      width="min(640px, 96vw)"
      header={{
        icon: PlusIcon,
        title: 'New custom template',
        subTitle: 'Choose where to add the template',
        onClose,
      }}
      footer={{
        children: (
          <DrawerSaveFooter $gap={12}>
            {error ? <Note status={StatusType.Error} message={error} fullWidth /> : null}
            <DrawerSaveFooterActions $gap={12}>
              <Button
                label="Cancel"
                variant={ButtonVariants.Secondary}
                size={ButtonSize.S}
                onClick={onClose}
                disabled={creatingAction}
              />
              <Button
                label="Continue"
                variant={ButtonVariants.Primary}
                size={ButtonSize.S}
                onClick={() => {
                  void handleContinue();
                }}
                disabled={!canContinue}
                loading={creatingAction}
              />
            </DrawerSaveFooterActions>
          </DrawerSaveFooter>
        ),
      }}
    >
      <DrawerBody $gap={16}>
        <OptionsColumn $gap={10}>
          <RadioCard
            data-id="url-templating-new-template-existing"
            title="Add to existing group"
            description="Recommended — reuse a scoped group to keep matching cheaper and reduce template fan-out."
            checked={path === 'existing'}
            onChange={(checked) => {
              if (checked) {
                setPath('existing');
              }
            }}
          />
          <RadioCard
            data-id="url-templating-new-template-new-group"
            title="Create a new group with specific scope"
            description={`Choose an action (defaults to "${UI_TEMPLATE_RULES_ACTION_NAME}"), then set namespace / workload / language scope.`}
            checked={path === 'new_group'}
            onChange={(checked) => {
              if (checked) {
                setPath('new_group');
              }
            }}
          />
          <RadioCard
            data-id="url-templating-new-template-entire-cluster"
            title="Add to entire cluster"
            description={`Great for evaluation and quick velocity — apply templates cluster-wide in "${UI_TEMPLATE_RULES_ACTION_NAME}".`}
            checked={path === 'entire_cluster'}
            onChange={(checked) => {
              if (checked) {
                setPath('entire_cluster');
              }
            }}
            footerNote={
              path === 'entire_cluster' ? (
                <Note
                  status={StatusType.Info}
                  message="Ideal when you want to try templates quickly across the cluster before narrowing scope."
                  fullWidth
                  smallIcon
                />
              ) : undefined
            }
          />
        </OptionsColumn>

        {path === 'existing' ? (
          <DetailPanel $gap={12}>
            <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
              Existing groups
            </Typography>
            <Search
              data-id="url-templating-new-template-group-search"
              value={groupQuery}
              onChange={setGroupQuery}
              placeholder="Search by scope, action, or template…"
              width="100%"
            />
            {filteredGroups.length === 0 ? (
              <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                {groups.length === 0
                  ? 'No rule groups yet. Create a new scoped group instead.'
                  : `No groups match "${groupQuery.trim()}".`}
              </Typography>
            ) : (
              <GroupList $gap={8}>
                {filteredGroups.map((group) => {
                  const key = groupKey(group);
                  const selected = selectedGroupKey === key;
                  return (
                    <GroupOption
                      key={key}
                      type="button"
                      $selected={selected}
                      onClick={() => setSelectedGroupKey(key)}
                    >
                      <FlexRow $gap={8} $alignItems="center" style={{ flexWrap: 'wrap' }}>
                        <Typography size={TypographySize.XS}>{group.actionLabel}</Typography>
                        <ActionOriginBadge uiGenerated={group.actionUiGenerated} />
                        {group.actionDisabled ? <Badge label="Disabled" /> : null}
                        <Badge label={`${group.templates.length} templates`} />
                      </FlexRow>
                      <ScopeTokens tokens={group.scopeTokens} />
                    </GroupOption>
                  );
                })}
              </GroupList>
            )}
          </DetailPanel>
        ) : null}

        {path === 'new_group' ? (
          <DetailPanel $gap={12}>
            <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
              Action
            </Typography>
            <DropDown
              data-id="url-templating-new-template-action"
              label="URL templatization action"
              options={actionOptions}
              values={selectedActionId ? [selectedActionId] : []}
              setValues={(values) => {
                const next = values[0] ?? UI_TEMPLATE_RULES_OPTION_ID;
                setSelectedActionId(next);
                if (next !== NEW_ACTION_OPTION_ID) {
                  setNewActionName('');
                }
              }}
              placeholder="Select action"
              width="100%"
            />
            {creatingNewAction ? (
              <Input
                data-id="url-templating-new-template-action-name"
                name="url-templating-new-template-action-name"
                label="New action name"
                value={newActionName}
                onChange={(event) => setNewActionName(event.target.value)}
                placeholder="e.g. Checkout service URL templates"
                width="100%"
                disabled={creatingAction}
              />
            ) : null}
            <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
              {`Defaults to "${UI_TEMPLATE_RULES_ACTION_NAME}". Actions can help keep templating rules organized, but they are optional — advanced users can create a separate action when useful.`}
            </Typography>
          </DetailPanel>
        ) : null}

        {path === 'entire_cluster' ? (
          <DetailPanel $gap={8}>
            <FlexRow $gap={8} $alignItems="center">
              <ActionIcon size={16} />
              <Typography size={TypographySize.XS}>{UI_TEMPLATE_RULES_ACTION_NAME}</Typography>
            </FlexRow>
            <ScopeTokens tokens={[{ type: 'entire_cluster' }]} />
            <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
              Continues in the edit drawer if an entire-cluster group already exists, otherwise creates one.
            </Typography>
          </DetailPanel>
        ) : null}
      </DrawerBody>
    </Drawer>
  );
}

function groupKey(group: CustomTemplateCrGroup): string {
  return `${group.actionId}:${group.groupIndex}`;
}
