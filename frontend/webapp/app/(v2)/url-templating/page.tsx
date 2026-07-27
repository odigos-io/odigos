'use client';

import React, { useMemo, useState, useCallback, type PropsWithChildren } from 'react';
import styled from 'styled-components';
import { ActionIcon } from '@odigos/ui-kit/icons';
import { RichTitle } from '@odigos/ui-kit/snippets';
import { useApiQuery } from '@odigos/ui-kit/contexts';
import { StatusType } from '@odigos/ui-kit/types';
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
  Typography,
  TypographyColor,
  TypographySize,
} from '@odigos/ui-kit/components';
import { CustomTemplateCrGroupsView, CustomTemplateListView, CustomTemplateTreeView } from './custom-templates-view';
import { CreateGroupTemplatesDrawer } from './create-group-templates-drawer';
import { DefaultTemplatizationCrGroupsView } from './default-templatization-view';
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
} from './url-templatization-aggregate';
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

export type CustomTemplateViewMode = 'list' | 'tree' | 'action';

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
  const [editTemplatesGroup, setEditTemplatesGroup] = useState<CustomTemplateCrGroup | null>(null);
  const [editFocusNewTemplate, setEditFocusNewTemplate] = useState(false);
  const [createGroupAction, setCreateGroupAction] = useState<Action | null>(null);
  const [createLockEntireClusterScope, setCreateLockEntireClusterScope] = useState(false);
  const [newTemplateOpen, setNewTemplateOpen] = useState(false);

  const urlTemplatizationActions = useMemo(() => {
    const actions = data?.computePlatform?.actions ?? [];
    return actions.filter(isUrlTemplatizationAction);
  }, [data]);

  const aggregation = useMemo(() => aggregateUrlTemplatization(urlTemplatizationActions), [urlTemplatizationActions]);

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
        <CenterThis>
          <NoData
            icon={ActionIcon}
            title="No URL templatization actions"
            subTitle="URL templatization actions configured in the cluster will appear here. Create a custom template to get started."
          />
        </CenterThis>
        {sharedDrawers}
      </PageShell>
    );
  }

  const { clusterDefaultEffect } = aggregation;

  const customSectionSummary = customTemplatesSectionSummary(aggregation.totalCustomRuleInstances);

  return (
    <PageShell headerAction={newTemplateButton}>
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
      >
        {defaultTemplatizationCrGroups.length === 0 ? (
          <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
            No default templatization scope groups configured. Built-in heuristic default
            templatization still applies when no custom template matches.
          </Typography>
        ) : (
          <DefaultTemplatizationCrGroupsView groups={defaultTemplatizationCrGroups} />
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
    </PageShell>
  );
}
