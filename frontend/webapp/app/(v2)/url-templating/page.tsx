'use client';

import React, { useMemo, useState, useCallback, type PropsWithChildren } from 'react';
import styled from 'styled-components';
import { ActionIcon } from '@odigos/ui-kit/icons';
import { RichTitle } from '@odigos/ui-kit/snippets';
import { useApiMutation, useApiQuery } from '@odigos/ui-kit/contexts';
import { ActionType, SignalType, StatusType } from '@odigos/ui-kit/types';
import type { Action } from '@odigos/ui-kit/types';
import {
  Button,
  ButtonSize,
  ButtonVariants,
  CenterThis,
  DataCard,
  FlexColumn,
  FlexRow,
  Loader,
  NoData,
  Note,
  Search,
  Segment,
  SegmentSize,
  SegmentVariant,
  StatusCard,
  Typography,
  TypographyColor,
  TypographySize,
  WarningModal,
} from '@odigos/ui-kit/components';
import { CustomTemplateCrGroupsView, CustomTemplateListView, CustomTemplateTreeView } from './custom-templates-view';
import { CreateGroupTemplatesDrawer } from './create-group-templates-drawer';
import { DefaultTemplatizationCrGroupsView } from './default-templatization-view';
import { DefaultTemplatizationDetailDrawer } from './default-templatization-detail-drawer';
import { EditDefaultTemplatizationDrawer } from './edit-default-templatization-drawer';
import { EditGroupTemplatesDrawer } from './edit-group-templates-drawer';
import { NewCustomTemplateDrawer, type NewCustomTemplateTarget } from './new-custom-template-drawer';
import { SectionHelpButton } from './section-help-button';
import { TemplateDetailDrawer } from './template-detail-drawer';
import type { UrlTemplateDetail } from './template-detail-types';
import {
  aggregateUrlTemplatization,
  collectCustomTemplateCrGroups,
  collectDefaultTemplatizationCrGroups,
  customTemplatesSectionSummary,
  isUrlTemplatizationAction,
  scopeTokensSortKey,
  type CustomTemplateCrGroup,
  type DefaultTemplatizationCrGroup,
} from './url-templatization-aggregate';
import {
  buildUpdateActionReenablingClusterWideDefaults,
  buildUpdateActionWithAppendedDefaultGroup,
} from './action-default-group-update';
import { findUiTemplateRulesAction, UI_TEMPLATE_RULES_ACTION_NAME } from './ui-template-rules';
import { useConfig } from '@/hooks';

const PageRoot = styled(FlexColumn)`
  flex: 1 1 0;
  align-self: stretch;
  width: 100%;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
`;

const PageScroll = styled(FlexColumn)`
  flex: 1;
  min-height: 0;
  width: 100%;
  padding: 24px;
  overflow-x: hidden;
  overflow-y: auto;
`;

const PageHeader = styled(FlexRow)`
  width: 100%;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
`;

const PageHeaderMain = styled.div`
  flex: 1;
  min-width: 240px;
`;

const CustomTemplatesToolbar = styled(FlexColumn)`
  width: 100%;
  gap: 12px;
  padding-bottom: 8px;
`;

const CustomTemplatesToolbarRow = styled(FlexRow)`
  width: 100%;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
`;

const CustomTemplatesSearchWrap = styled.div`
  flex: 1;
  min-width: 200px;
`;

const StatusActions = styled(FlexRow)`
  width: 100%;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
`;

const StatusBannerWrap = styled.div`
  flex: 0 0 auto;
  width: 100%;
  align-self: stretch;

  /* StatusCard uses flex: 1; keep the page banner content-sized. */
  & > * {
    flex: 0 0 auto;
  }
`;

const StatusPanel = styled(FlexColumn)<{ $status: StatusType }>`
  flex: 0 0 auto;
  width: 100%;
  gap: 12px;
  padding: 16px 18px;
  border-radius: 12px;
  border: 1px solid
    ${({ theme, $status }) => {
      if ($status === StatusType.Warning) {
        return theme.v2.colors.yellow[600];
      }
      if ($status === StatusType.Success) {
        return theme.v2.colors.green[600];
      }
      if ($status === StatusType.Info) {
        return theme.v2.colors.blue[500];
      }
      return theme.v2.colors.silver[600];
    }};
  background: ${({ theme, $status }) => {
    if ($status === StatusType.Warning) {
      return `color-mix(in srgb, ${theme.v2.colors.yellow[600]} 10%, ${theme.v2.colors.silver[900]})`;
    }
    if ($status === StatusType.Success) {
      return `color-mix(in srgb, ${theme.v2.colors.green[600]} 10%, ${theme.v2.colors.silver[900]})`;
    }
    if ($status === StatusType.Info) {
      return `color-mix(in srgb, ${theme.v2.colors.blue[500]} 10%, ${theme.v2.colors.silver[900]})`;
    }
    return theme.v2.colors.silver[900];
  }};
`;

export type CustomTemplateViewMode = 'list' | 'tree' | 'action';

const ENABLE_CLUSTER_DEFAULT_WARNING =
  'Default heuristic templating works for most URLs, but not always. Enabling it for the entire cluster can increase cardinality when unusual paths are templatized incorrectly.';

function TemplatingStatusBanner({
  status,
  title,
  description,
  action,
}: {
  status: StatusType;
  title: string;
  description: string;
  action?: React.ReactNode;
}) {
  if (!action) {
    return (
      <StatusBannerWrap>
        <StatusCard status={status} title={title} description={description} />
      </StatusBannerWrap>
    );
  }

  return (
    <StatusPanel $status={status} $gap={12}>
      <FlexColumn $gap={4}>
        <Typography size={TypographySize.XS}>{title}</Typography>
        <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
          {description}
        </Typography>
      </FlexColumn>
      <StatusActions $gap={12}>{action}</StatusActions>
    </StatusPanel>
  );
}

function pageSummaryStatus(
  effect: { mode: string; disabledClusterWide: boolean },
  enabledCustomRuleCount: number,
  enabledActionCount: number,
): StatusType {
  if (enabledActionCount === 0) {
    return StatusType.Disabled;
  }
  const defaultIsOff = effect.disabledClusterWide || effect.mode === 'none';
  if (defaultIsOff && enabledCustomRuleCount === 0) {
    return StatusType.Disabled;
  }
  if (defaultIsOff) {
    return StatusType.Warning;
  }
  if (effect.mode === 'scoped') {
    return StatusType.Info;
  }
  return StatusType.Success;
}

function PageShell({
  children,
  headerAction,
}: PropsWithChildren<{ headerAction?: React.ReactNode }>) {
  return (
    <PageRoot>
      <PageScroll $gap={20}>
        <PageHeader $gap={16}>
          <PageHeaderMain>
            <RichTitle
              icon={ActionIcon}
              iconWithWrapper
              title="URL Templating"
              subTitle="Custom URL templates and default templatization scopes across URL templatization actions."
            />
          </PageHeaderMain>
          {headerAction}
        </PageHeader>
        {children}
      </PageScroll>
    </PageRoot>
  );
}

export default function UrlTemplatingPage() {
  const { isReadonly } = useConfig();
  const { data, loading, error, unsupported, refetch } = useApiQuery('GET_ACTIONS');
  const [customTemplateQuery, setCustomTemplateQuery] = useState('');
  const [customTemplateView, setCustomTemplateView] = useState<CustomTemplateViewMode>('action');
  const [selectedTemplateDetail, setSelectedTemplateDetail] = useState<UrlTemplateDetail | null>(null);
  const [selectedDefaultGroup, setSelectedDefaultGroup] = useState<DefaultTemplatizationCrGroup | null>(null);
  const [editDefaultGroup, setEditDefaultGroup] = useState<DefaultTemplatizationCrGroup | null>(null);
  const [createDefaultAction, setCreateDefaultAction] = useState<Action | null>(null);
  const [editTemplatesGroup, setEditTemplatesGroup] = useState<CustomTemplateCrGroup | null>(null);
  const [editFocusNewTemplate, setEditFocusNewTemplate] = useState(false);
  const [createGroupAction, setCreateGroupAction] = useState<Action | null>(null);
  const [createLockEntireClusterScope, setCreateLockEntireClusterScope] = useState(false);
  const [newTemplateOpen, setNewTemplateOpen] = useState(false);
  const [confirmEnableClusterDefaultOpen, setConfirmEnableClusterDefaultOpen] = useState(false);
  const [enableClusterDefaultError, setEnableClusterDefaultError] = useState<string | null>(null);

  const [createAction, { loading: creatingAction }] = useApiMutation('CREATE_ACTION', {
    refetchQueries: ['GetActions'],
    awaitRefetchQueries: true,
  });
  const [updateAction, { loading: updatingAction }] = useApiMutation('UPDATE_ACTION', {
    refetchQueries: ['GetActions'],
    awaitRefetchQueries: true,
  });
  const enablingClusterDefault = creatingAction || updatingAction;

  const urlTemplatizationActions = useMemo(() => {
    const actions = data?.computePlatform?.actions ?? [];
    return actions.filter(isUrlTemplatizationAction);
  }, [data]);

  const aggregation = useMemo(() => aggregateUrlTemplatization(urlTemplatizationActions), [urlTemplatizationActions]);

  const defaultTemplatizationOff =
    aggregation.enabledActionCount === 0 ||
    aggregation.clusterDefaultEffect.mode === 'none' ||
    aggregation.clusterDefaultEffect.disabledClusterWide;
  const filteredCustomTemplates = useMemo(() => {
    const query = customTemplateQuery.trim().toLowerCase();
    if (!query) {
      return aggregation.customTemplates;
    }
    return aggregation.customTemplates.filter((item) => item.template.toLowerCase().includes(query));
  }, [aggregation.customTemplates, customTemplateQuery]);

  const customTemplateCrGroups = useMemo(
    () => collectCustomTemplateCrGroups(urlTemplatizationActions),
    [urlTemplatizationActions],
  );

  const defaultTemplatizationCrGroups = useMemo(
    () => collectDefaultTemplatizationCrGroups(urlTemplatizationActions),
    [urlTemplatizationActions],
  );

  const editTemplatesAction = useMemo(() => {
    if (!editTemplatesGroup) {
      return null;
    }
    return urlTemplatizationActions.find((action) => action.id === editTemplatesGroup.actionId) ?? null;
  }, [editTemplatesGroup, urlTemplatizationActions]);

  const selectedTemplateAction = useMemo(() => {
    if (!selectedTemplateDetail) {
      return null;
    }
    return urlTemplatizationActions.find((action) => action.id === selectedTemplateDetail.actionId) ?? null;
  }, [selectedTemplateDetail, urlTemplatizationActions]);

  const selectedDefaultAction = useMemo(() => {
    if (createDefaultAction) {
      return createDefaultAction;
    }
    const group = editDefaultGroup ?? selectedDefaultGroup;
    if (!group) {
      return null;
    }
    return urlTemplatizationActions.find((action) => action.id === group.actionId) ?? null;
  }, [createDefaultAction, editDefaultGroup, selectedDefaultGroup, urlTemplatizationActions]);

  const handleOpenCreateDefaultRule = useCallback(async () => {
    const enabledActions = urlTemplatizationActions.filter((action) => !action.disabled);
    let target = findUiTemplateRulesAction(enabledActions) ?? enabledActions[0] ?? null;
    if (!target) {
      const { data: created, error: createError } = await createAction({
        action: {
          type: ActionType.URLTemplatization,
          name: UI_TEMPLATE_RULES_ACTION_NAME,
          notes: 'Default action for URL templates created from the Odigos UI.',
          disabled: false,
          signals: [SignalType.Traces],
          fields: {
            urlTemplatizationRulesGroups: [],
            urlTemplatizationDefaultGroups: [],
          },
        },
      });
      if (createError) {
        return;
      }
      if (!created?.createAction) {
        return;
      }
      target = created.createAction;
      void refetch();
    }
    setEditDefaultGroup(null);
    setCreateDefaultAction(target);
  }, [createAction, refetch, urlTemplatizationActions]);

  const drawerActionRuleGroups = useMemo(() => {
    if (!selectedTemplateDetail) {
      return [];
    }
    return customTemplateCrGroups
      .filter((group) => group.actionId === selectedTemplateDetail.actionId)
      .sort((a, b) => a.groupIndex - b.groupIndex);
  }, [selectedTemplateDetail, customTemplateCrGroups]);

  const handleSelectTemplate = useCallback(
    (detail: UrlTemplateDetail) => {
      if (detail.groupIndex === undefined) {
        const group = customTemplateCrGroups.find(
          (g) =>
            g.actionId === detail.actionId &&
            g.templates.includes(detail.template) &&
            scopeTokensSortKey(g.scopeTokens) === scopeTokensSortKey(detail.scopeTokens),
        );
        if (group) {
          setSelectedTemplateDetail({
            ...detail,
            groupIndex: group.groupIndex,
            groupNotes: group.groupNotes,
          });
          return;
        }
      }
      setSelectedTemplateDetail(detail);
    },
    [customTemplateCrGroups],
  );

  const handleEditGroupTemplates = useCallback(
    (group: CustomTemplateCrGroup, options?: { focusNewTemplate?: boolean }) => {
      const fullGroup =
        customTemplateCrGroups.find(
          (candidate) => candidate.actionId === group.actionId && candidate.groupIndex === group.groupIndex,
        ) ?? group;
      setEditFocusNewTemplate(!!options?.focusNewTemplate);
      setEditTemplatesGroup(fullGroup);
    },
    [customTemplateCrGroups],
  );

  const handleNewCustomTemplateContinue = useCallback(
    (target: NewCustomTemplateTarget) => {
      if (target.kind === 'edit_group') {
        handleEditGroupTemplates(target.group, { focusNewTemplate: true });
        return;
      }
      setCreateLockEntireClusterScope(!!target.lockEntireClusterScope);
      setCreateGroupAction(target.action);
    },
    [handleEditGroupTemplates],
  );

  const handleEnableClusterWideDefault = useCallback(async () => {
    setEnableClusterDefaultError(null);
    try {
      const enabledActions = urlTemplatizationActions.filter((action) => !action.disabled);
      let reenabledAny = false;
      for (const action of enabledActions) {
        const variables = buildUpdateActionReenablingClusterWideDefaults(action);
        if (!variables) {
          continue;
        }
        const { error: updateError } = await updateAction(variables);
        if (updateError) {
          throw new Error(updateError.message || 'Failed to enable default templating');
        }
        reenabledAny = true;
      }
      if (reenabledAny) {
        setConfirmEnableClusterDefaultOpen(false);
        void refetch();
        return;
      }

      let target = findUiTemplateRulesAction(enabledActions) ?? enabledActions[0];
      if (!target) {
        const { data: created, error: createError } = await createAction({
          action: {
            type: ActionType.URLTemplatization,
            name: UI_TEMPLATE_RULES_ACTION_NAME,
            notes: 'Default action for URL templates created from the Odigos UI.',
            disabled: false,
            signals: [SignalType.Traces],
            fields: {
              urlTemplatizationRulesGroups: [],
              urlTemplatizationDefaultGroups: [
                {
                  scopes: null,
                  disabled: false,
                  skipPolicy: null,
                },
              ],
            },
          },
        });
        if (createError) {
          throw new Error(createError.message || 'Failed to enable default templating');
        }
        if (!created?.createAction) {
          throw new Error('Failed to enable default templating');
        }
        setConfirmEnableClusterDefaultOpen(false);
        void refetch();
        return;
      }

      const { error: updateError } = await updateAction(buildUpdateActionWithAppendedDefaultGroup(target));
      if (updateError) {
        throw new Error(updateError.message || 'Failed to enable default templating');
      }
      setConfirmEnableClusterDefaultOpen(false);
      void refetch();
    } catch (err) {
      setEnableClusterDefaultError(
        err instanceof Error ? err.message : 'Failed to enable default templating',
      );
    }
  }, [createAction, refetch, updateAction, urlTemplatizationActions]);

  const enableClusterDefaultButton =
    isReadonly || !defaultTemplatizationOff ? null : (
      <Button
        data-id="url-templating-enable-cluster-default"
        label="Enable default templating for entire cluster"
        variant={ButtonVariants.Primary}
        size={ButtonSize.S}
        onClick={() => {
          setEnableClusterDefaultError(null);
          setConfirmEnableClusterDefaultOpen(true);
        }}
        disabled={enablingClusterDefault}
      />
    );

  const enableClusterDefaultModal = (
    <WarningModal
      isOpen={confirmEnableClusterDefaultOpen}
      title="Enable default templating cluster-wide?"
      description={ENABLE_CLUSTER_DEFAULT_WARNING}
      onClose={() => {
        if (!enablingClusterDefault) {
          setConfirmEnableClusterDefaultOpen(false);
          setEnableClusterDefaultError(null);
        }
      }}
      denyButton={{
        label: 'Cancel',
        onClick: () => {
          setConfirmEnableClusterDefaultOpen(false);
          setEnableClusterDefaultError(null);
        },
        disabled: enablingClusterDefault,
      }}
      approveButton={{
        label: 'Enable for entire cluster',
        onClick: () => {
          void handleEnableClusterWideDefault();
        },
        loading: enablingClusterDefault,
        disabled: enablingClusterDefault,
      }}
    >
      {enableClusterDefaultError ? (
        <Note status={StatusType.Error} message={enableClusterDefaultError} fullWidth />
      ) : null}
    </WarningModal>
  );

  const filteredCustomTemplateCrGroups = useMemo(() => {
    const query = customTemplateQuery.trim().toLowerCase();
    if (!query) {
      return customTemplateCrGroups;
    }
    return customTemplateCrGroups
      .map((group) => ({
        ...group,
        templates: group.templates.filter((template) => template.toLowerCase().includes(query)),
      }))
      .filter((group) => group.templates.length > 0);
  }, [customTemplateCrGroups, customTemplateQuery]);

  const filteredCustomTemplateCount = useMemo(() => {
    if (customTemplateView === 'action') {
      return filteredCustomTemplateCrGroups.reduce((sum, group) => sum + group.templates.length, 0);
    }
    return filteredCustomTemplates.length;
  }, [customTemplateView, filteredCustomTemplateCrGroups, filteredCustomTemplates.length]);

  const newTemplateButton = isReadonly ? null : (
    <Button
      data-id="url-templating-new-custom-template"
      label="+ New custom template"
      variant={ButtonVariants.Primary}
      size={ButtonSize.S}
      onClick={() => setNewTemplateOpen(true)}
    />
  );

  const newDefaultRuleButton = isReadonly ? null : (
    <Button
      data-id="url-templating-new-default-rule"
      label="+ New default rule"
      variant={ButtonVariants.Primary}
      size={ButtonSize.S}
      onClick={() => {
        void handleOpenCreateDefaultRule();
      }}
      disabled={enablingClusterDefault}
    />
  );

  const sharedDrawers = (
    <>
      <NewCustomTemplateDrawer
        isOpen={newTemplateOpen}
        actions={urlTemplatizationActions}
        groups={customTemplateCrGroups}
        onClose={() => setNewTemplateOpen(false)}
        onContinue={handleNewCustomTemplateContinue}
      />
      <TemplateDetailDrawer
        detail={selectedTemplateDetail}
        action={isReadonly ? null : selectedTemplateAction}
        actionRuleGroups={drawerActionRuleGroups}
        onSelectTemplate={handleSelectTemplate}
        onEditGroupTemplates={
          isReadonly
            ? undefined
            : (group) => {
                setSelectedTemplateDetail(null);
                handleEditGroupTemplates(group);
              }
        }
        onDeleted={
          isReadonly
            ? undefined
            : () => {
                void refetch();
              }
        }
        onClose={() => setSelectedTemplateDetail(null)}
      />
      <DefaultTemplatizationDetailDrawer
        group={selectedDefaultGroup}
        action={isReadonly ? null : selectedDefaultAction}
        onEdit={
          isReadonly
            ? undefined
            : (group) => {
                setSelectedDefaultGroup(null);
                setEditDefaultGroup(group);
              }
        }
        onDeleted={
          isReadonly
            ? undefined
            : () => {
                void refetch();
              }
        }
        onClose={() => setSelectedDefaultGroup(null)}
      />
      <EditDefaultTemplatizationDrawer
        group={editDefaultGroup}
        action={isReadonly ? null : selectedDefaultAction}
        creating={!!createDefaultAction && !editDefaultGroup}
        onClose={() => {
          setEditDefaultGroup(null);
          setCreateDefaultAction(null);
        }}
        onSaved={() => {
          void refetch();
        }}
      />
      <EditGroupTemplatesDrawer
        group={editTemplatesGroup}
        action={editTemplatesAction}
        focusNewTemplate={editFocusNewTemplate}
        onClose={() => {
          setEditTemplatesGroup(null);
          setEditFocusNewTemplate(false);
        }}
        onSaved={() => {
          void refetch();
        }}
      />
      <CreateGroupTemplatesDrawer
        action={createGroupAction}
        lockEntireClusterScope={createLockEntireClusterScope}
        onClose={() => {
          setCreateGroupAction(null);
          setCreateLockEntireClusterScope(false);
        }}
        onSaved={() => {
          void refetch();
        }}
      />
    </>
  );

  if (unsupported) {
    return (
      <PageShell>
        <Note status={StatusType.Warning} message="Actions are not available in this environment." fullWidth />
      </PageShell>
    );
  }

  if (loading && !data) {
    return (
      <PageShell>
        <CenterThis>
          <Loader withSpinnerOld scaleSpinnerOld={2} />
        </CenterThis>
      </PageShell>
    );
  }

  if (error) {
    return (
      <PageShell>
        <Note status={StatusType.Error} message={error.message || 'Failed to load actions'} fullWidth />
      </PageShell>
    );
  }

  if (!urlTemplatizationActions.length) {
    return (
      <PageShell headerAction={newTemplateButton}>
        <TemplatingStatusBanner
          status={StatusType.Disabled}
          title="URL templating is OFF"
          description="No URL templatization actions are configured in this cluster."
          action={enableClusterDefaultButton}
        />
        <CenterThis>
          <NoData
            icon={ActionIcon}
            title="No URL templatization actions"
            subTitle="URL templatization actions configured in the cluster will appear here. Create a custom template to get started."
          />
        </CenterThis>
        {sharedDrawers}
        {enableClusterDefaultModal}
      </PageShell>
    );
  }

  const { clusterDefaultEffect, pageSummary } = aggregation;

  const customSectionSummary = customTemplatesSectionSummary(aggregation.totalCustomRuleInstances);

  return (
    <PageShell headerAction={newTemplateButton}>
      <TemplatingStatusBanner
        status={pageSummaryStatus(
          clusterDefaultEffect,
          aggregation.enabledCustomRuleInstances,
          aggregation.enabledActionCount,
        )}
        title={pageSummary}
        description={clusterDefaultEffect.description}
        action={enableClusterDefaultButton}
      />
      <DataCard
        bgTint="800"
        withCollapse
        collapseIsDefaultOpen
        richTitle={{
          title: 'Default templatization',
          subTitle: clusterDefaultEffect.sectionSummary,
          children: (
            <SectionHelpButton
              data-id="url-templating-default-help"
              title="Default templatization"
              text="Default templatization is the built-in fallback that turns concrete URL paths into templates when no custom template matches. Use it to control heuristic behavior (enable, disable, or skip policies) for selected scopes."
            />
          ),
        }}
        renderOnRightSide={newDefaultRuleButton}
      >
        {defaultTemplatizationCrGroups.length === 0 ? (
          <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
            No default templatization rules configured.
          </Typography>
        ) : (
          <DefaultTemplatizationCrGroupsView
            groups={defaultTemplatizationCrGroups}
            onSelectGroup={setSelectedDefaultGroup}
            onEditGroup={
              isReadonly
                ? undefined
                : (group) => {
                    setCreateDefaultAction(null);
                    setEditDefaultGroup(group);
                  }
            }
          />
        )}
      </DataCard>

      <DataCard
        bgTint="800"
        withCollapse
        collapseIsDefaultOpen
        richTitle={{
          title: 'Custom templates',
          subTitle: customSectionSummary,
          children: (
            <SectionHelpButton
              data-id="url-templating-custom-help"
              title="Custom templates"
              text="Custom templates are explicit URL patterns you define (for example /users/{id}). They take precedence over default templatization and are grouped by scope so matching stays targeted and efficient."
            />
          ),
        }}
      >
        {aggregation.customTemplates.length === 0 ? (
          <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
            No custom template rules configured.
          </Typography>
        ) : (
          <>
            <CustomTemplatesToolbar $gap={12}>
              <CustomTemplatesToolbarRow $gap={12}>
                <CustomTemplatesSearchWrap>
                  <Search
                    data-id="url-templating-custom-template-search"
                    value={customTemplateQuery}
                    onChange={setCustomTemplateQuery}
                    placeholder="Search templates…"
                    width="100%"
                  />
                </CustomTemplatesSearchWrap>
                <Segment<CustomTemplateViewMode>
                  data-id="url-templating-custom-template-view"
                  size={SegmentSize.S}
                  variant={SegmentVariant.Filled}
                  selected={customTemplateView}
                  setSelected={setCustomTemplateView}
                  options={[
                    { value: 'action', label: 'Groups' },
                    { value: 'tree', label: 'Tree' },
                    { value: 'list', label: 'List' },
                  ]}
                />
              </CustomTemplatesToolbarRow>
              {customTemplateQuery.trim() ? (
                <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                  {filteredCustomTemplateCount} of {aggregation.totalCustomRuleInstances} templates
                </Typography>
              ) : null}
            </CustomTemplatesToolbar>
            {filteredCustomTemplateCount === 0 ? (
              <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                {`No templates match "${customTemplateQuery.trim()}".`}
              </Typography>
            ) : customTemplateView === 'tree' ? (
              <CustomTemplateTreeView items={filteredCustomTemplates} onSelectTemplate={handleSelectTemplate} />
            ) : customTemplateView === 'action' ? (
              <CustomTemplateCrGroupsView
                groups={filteredCustomTemplateCrGroups}
                onSelectTemplate={handleSelectTemplate}
                onEditGroupTemplates={isReadonly ? undefined : handleEditGroupTemplates}
                onCreateGroup={
                  isReadonly
                    ? undefined
                    : (actionId) => {
                        const action = urlTemplatizationActions.find((candidate) => candidate.id === actionId);
                        if (action) {
                          setCreateLockEntireClusterScope(false);
                          setCreateGroupAction(action);
                        }
                      }
                }
              />
            ) : (
              <CustomTemplateListView items={filteredCustomTemplates} onSelectTemplate={handleSelectTemplate} />
            )}
          </>
        )}
      </DataCard>
      {sharedDrawers}
      {enableClusterDefaultModal}
    </PageShell>
  );
}
