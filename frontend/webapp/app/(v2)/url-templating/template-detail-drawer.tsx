'use client';

import React, { useEffect, useMemo, useState } from 'react';
import styled from 'styled-components';
import { ActionIcon, ChevronRightIcon, EditIcon, TrashIcon } from '@odigos/ui-kit/icons';
import type { Action } from '@odigos/ui-kit/types';
import { StatusType } from '@odigos/ui-kit/types';
import {
  Divider,
  Drawer,
  FlexColumn,
  FlexRow,
  IconButton,
  IconButtonSize,
  Note,
  Tooltip,
  Typography,
  TypographyColor,
  TypographySize,
  WarningModal,
} from '@odigos/ui-kit/components';
import { useApiMutation } from '@odigos/ui-kit/contexts';
import { ActionOriginBadge, actionOriginTitleBadge } from './action-origin-badge';
import { ScopeTokens } from './scope-tokens';
import { parseTemplatePath, TemplatePath, TemplatePathSegmentLabel } from './template-path';
import { sortUrlTemplates } from './template-sort';
import { scopeTokensSortKey } from './scope-token-types';
import type { UrlTemplateDetail } from './template-detail-types';
import {
  detailFromCrGroup,
  detailFromCrTemplate,
  isActiveRuleGroup,
  type CrTemplateGroupRef,
} from './template-detail-types';
import { TemplatePathSimulator } from './template-path-simulator';
import { templateRowActivationProps } from './template-row-activation';
import { buildUpdateActionWithGroupTemplates } from './action-group-templates-update';

const DrawerBody = styled(FlexColumn)`
  width: 100%;
  padding: 0 24px 24px;
  gap: 16px;
`;

const DetailBlock = styled(FlexColumn)`
  width: 100%;
  gap: 8px;
`;

const FieldList = styled(FlexColumn)`
  width: 100%;
  gap: 10px;
`;

const Field = styled(FlexColumn)`
  width: 100%;
  gap: 2px;
`;

const ResourceName = styled.span`
  font-family: ${({ theme }) => theme.font_family.code ?? theme.font_family.primary};
  font-size: ${({ theme }) => theme.v2.text.size.xs}px;
  line-height: 1.4;
  color: ${({ theme }) => theme.v2.colors.white[500]};
  word-break: break-word;
`;

const SegmentTrack = styled(FlexRow)`
  width: 100%;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 0;
  row-gap: 8px;
`;

const SegmentCell = styled(FlexColumn)`
  align-items: center;
  gap: 4px;
  min-width: 0;
  max-width: 100%;
`;

const SegmentSlash = styled.span`
  flex-shrink: 0;
  align-self: flex-start;
  padding: 0 2px;
  font-family: ${({ theme }) => theme.font_family.primary};
  font-size: 12px;
  line-height: 1.4;
  color: ${({ theme }) => theme.v2.colors.white[500]};
`;

const RuleGroupRow = styled(FlexColumn)<{ $active?: boolean }>`
  width: 100%;
  gap: 6px;
  padding: 8px 6px;
  margin: 0 -6px;
  border-radius: 6px;
  cursor: pointer;
  border: 1px solid
    ${({ theme, $active }) => ($active ? theme.v2.colors.silver[600] : 'transparent')};
  background: ${({ theme, $active }) => ($active ? theme.v2.colors.silver[900] : 'transparent')};
  transition: background 0.12s ease;

  &:hover {
    background: ${({ theme }) => theme.v2.colors.silver[800]};
  }

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.v2.colors.blue[400]};
    outline-offset: 2px;
  }
`;

const RuleGroupList = styled(FlexColumn)`
  width: 100%;
  gap: 4px;
`;

const GroupTemplateRow = styled(FlexRow)`
  width: 100%;
  align-items: center;
  padding: 6px 6px;
  margin: 0 -6px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.12s ease;

  &:hover {
    background: ${({ theme }) => theme.v2.colors.silver[800]};
  }

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.v2.colors.blue[400]};
    outline-offset: 2px;
  }
`;

const GroupTemplateList = styled(FlexColumn)`
  width: 100%;
  gap: 2px;
`;

const CollapseHeader = styled(FlexRow)`
  width: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 4px 0;
  cursor: pointer;
  user-select: none;

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.v2.colors.blue[400]};
    outline-offset: 2px;
    border-radius: 4px;
  }
`;

const CollapseChevron = styled.span<{ $open: boolean }>`
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  transform: rotate(${({ $open }) => ($open ? '90deg' : '0deg')});
  transition: transform 0.15s ease;
  color: ${({ theme }) => theme.v2.colors.grey[400]};
`;

const CollapseBody = styled(FlexColumn)`
  width: 100%;
  gap: 14px;
  padding-top: 8px;
`;

const segmentKindLabel: Record<string, string> = {
  static: 'Exact',
  variable: 'Variable',
  wildcard: 'Any',
};

type TemplateDetailDrawerProps = {
  detail: UrlTemplateDetail | null;
  action: Action | null;
  actionRuleGroups: CrTemplateGroupRef[];
  onSelectTemplate: (detail: UrlTemplateDetail) => void;
  onEditGroupTemplates?: (group: CrTemplateGroupRef) => void;
  onDeleted?: () => void;
  onClose: () => void;
};

function DetailField({
  label,
  labelTooltip,
  children,
}: {
  label: string;
  labelTooltip?: string;
  children: React.ReactNode;
}) {
  const labelNode = (
    <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
      {label}
    </Typography>
  );

  return (
    <Field $gap={2}>
      {labelTooltip ? (
        <Tooltip text={labelTooltip} inline>
          {labelNode}
        </Tooltip>
      ) : (
        labelNode
      )}
      {children}
    </Field>
  );
}

function actionDisplayNameForDrawer(detail: UrlTemplateDetail): string | null {
  const name = detail.actionDisplayName?.trim();
  if (!name || name === detail.actionId) {
    return null;
  }
  return name;
}

function DrawerCollapseSection({
  title,
  summary,
  defaultOpen = false,
  titleAction,
  children,
}: {
  title: string;
  summary?: string;
  defaultOpen?: boolean;
  titleAction?: React.ReactNode;
  children: React.ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <DetailBlock $gap={0}>
      <CollapseHeader
        $gap={12}
        role="button"
        tabIndex={0}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            setOpen((value) => !value);
          }
        }}
      >
        <FlexColumn $gap={2} style={{ flex: 1, minWidth: 0 }}>
          <FlexRow $gap={8} $alignItems="center">
            <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
              {title}
            </Typography>
            {titleAction}
          </FlexRow>
          {!open && summary ? (
            <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
              {summary}
            </Typography>
          ) : null}
        </FlexColumn>
        <CollapseChevron $open={open}>
          <ChevronRightIcon size={14} />
        </CollapseChevron>
      </CollapseHeader>
      {open ? <CollapseBody $gap={14}>{children}</CollapseBody> : null}
    </DetailBlock>
  );
}

export function TemplateDetailDrawer({
  detail,
  action,
  actionRuleGroups,
  onSelectTemplate,
  onEditGroupTemplates,
  onDeleted,
  onClose,
}: TemplateDetailDrawerProps) {
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const [updateAction, { loading: deleting }] = useApiMutation('UPDATE_ACTION', {
    refetchQueries: ['GetActions'],
    awaitRefetchQueries: true,
  });

  useEffect(() => {
    setConfirmDeleteOpen(false);
    setDeleteError(null);
  }, [detail]);

  const segments = useMemo(
    () => (detail?.template.trim() ? parseTemplatePath(detail.template) : []),
    [detail],
  );

  const sortedRuleGroups = useMemo(
    () => [...actionRuleGroups].sort((a, b) => a.groupIndex - b.groupIndex),
    [actionRuleGroups],
  );

  const activeRuleGroup = useMemo(() => {
    if (!detail) {
      return undefined;
    }
    const byIndex = sortedRuleGroups.find((group) => isActiveRuleGroup(detail, group));
    if (byIndex) {
      return byIndex;
    }
    if (!detail.template.trim()) {
      return undefined;
    }
    return sortedRuleGroups.find(
      (group) =>
        group.actionId === detail.actionId &&
        group.templates.includes(detail.template) &&
        scopeTokensSortKey(group.scopeTokens) === scopeTokensSortKey(detail.scopeTokens),
    );
  }, [sortedRuleGroups, detail]);

  const otherGroupTemplates = useMemo(() => {
    if (!activeRuleGroup) {
      return [];
    }
    if (!detail?.template.trim()) {
      return sortUrlTemplates(activeRuleGroup.templates);
    }
    return sortUrlTemplates(activeRuleGroup.templates.filter((template) => template !== detail.template));
  }, [activeRuleGroup, detail?.template]);

  const siblingTemplatesSummary = useMemo(() => {
    const count = otherGroupTemplates.length;
    if (count === 0) {
      return undefined;
    }
    return `${count} ${count === 1 ? 'template' : 'templates'}`;
  }, [otherGroupTemplates.length]);

  const headerBadges = useMemo(() => {
    if (!detail) {
      return undefined;
    }
    const badges = [actionOriginTitleBadge(detail.actionUiGenerated)];
    if (detail.actionDisabled) {
      badges.push({ label: 'Disabled' });
    }
    if (detail.count > 1) {
      badges.push({ label: `×${detail.count}` });
    }
    return badges;
  }, [detail]);

  const drawerDisplayName = detail ? actionDisplayNameForDrawer(detail) : null;

  const groupCollapseSummary = useMemo(() => {
    if (!detail || sortedRuleGroups.length <= 1) {
      return undefined;
    }
    const activeIndex = activeRuleGroup?.groupIndex;
    if (activeIndex === undefined) {
      return undefined;
    }
    return `This template is in group ${activeIndex + 1}`;
  }, [detail, sortedRuleGroups.length, activeRuleGroup?.groupIndex]);

  const showGroupSection = sortedRuleGroups.length > 1;

  const canDeleteTemplate = !!action && !!activeRuleGroup && !!detail?.template.trim() && !!onDeleted;

  const handleDeleteTemplate = async () => {
    if (!detail || !action || !activeRuleGroup || !onDeleted) {
      return;
    }
    setDeleteError(null);
    try {
      const remaining = activeRuleGroup.templates.filter((template) => template !== detail.template);
      const variables = buildUpdateActionWithGroupTemplates(action, activeRuleGroup.groupIndex, remaining);
      const { error } = await updateAction(variables);
      if (error) {
        setDeleteError(error.message || 'Failed to delete template');
        return;
      }
      setConfirmDeleteOpen(false);
      onDeleted();
      onClose();
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Failed to delete template');
    }
  };

  return (
    <>
    <Drawer
      isOpen={!!detail}
      width="min(720px, 96vw)"
      header={
        detail
          ? {
              icon: ActionIcon,
              title: 'URL template',
              subTitle: detail.actionLabel,
              titleBadges: headerBadges,
              actions: canDeleteTemplate
                ? [
                    {
                      id: 'delete-template',
                      label: 'Delete template',
                      icon: TrashIcon,
                      onClick: () => {
                        setDeleteError(null);
                        setConfirmDeleteOpen(true);
                      },
                    },
                  ]
                : undefined,
              onClose,
            }
          : undefined
      }
    >
      {detail ? (
        <DrawerBody $gap={16}>
          <DetailBlock $gap={8}>
            <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
              Template
            </Typography>
            {detail.template.trim() ? (
              <TemplatePath template={detail.template} disabled={detail.actionDisabled} fontSize={14} />
            ) : (
              <Typography size={TypographySize.XS} color={TypographyColor.Secondary}>
                No templates in this rule group
              </Typography>
            )}
          </DetailBlock>

          <Divider />

          <DetailBlock $gap={8}>
            <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
              Action
            </Typography>
            <FieldList $gap={10}>
              {drawerDisplayName ? (
                <DetailField label="Display name">
                  <Typography size={TypographySize.XS}>{drawerDisplayName}</Typography>
                </DetailField>
              ) : null}
              <DetailField label="Resource name" labelTooltip="Action CR metadata.name">
                <ResourceName>{detail.actionId}</ResourceName>
              </DetailField>
              <DetailField label="Origin">
                <ActionOriginBadge uiGenerated={detail.actionUiGenerated} />
              </DetailField>
            </FieldList>
          </DetailBlock>

          {segments.length ? (
            <>
              <Divider />
              <DetailBlock $gap={8}>
                <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                  Path segments
                </Typography>
                <SegmentTrack $gap={0}>
                  {detail.template.trimStart().startsWith('/') ? <SegmentSlash>/</SegmentSlash> : null}
                  {segments.map((segment, index) => (
                    <React.Fragment key={`${index}-${segment.text}`}>
                      {index > 0 ? <SegmentSlash>/</SegmentSlash> : null}
                      <SegmentCell $gap={4}>
                        <TemplatePathSegmentLabel kind={segment.kind} text={segment.text} fontSize={12} />
                        <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                          {segmentKindLabel[segment.kind] ?? segment.kind}
                        </Typography>
                      </SegmentCell>
                    </React.Fragment>
                  ))}
                </SegmentTrack>
                <TemplatePathSimulator template={detail.template} actionDisabled={detail.actionDisabled} />
              </DetailBlock>
            </>
          ) : null}

          {showGroupSection ? (
            <>
              <Divider />
              <DrawerCollapseSection
                title="Switch rule group"
                summary={groupCollapseSummary}
                defaultOpen={false}
              >
                <RuleGroupList $gap={4}>
                  {sortedRuleGroups.map((group) => {
                    const active = isActiveRuleGroup(detail, group);
                    const openGroup = () => onSelectTemplate(detailFromCrGroup(group, detail.template));
                    const groupKey = `${group.actionId}:${group.groupIndex}:${scopeTokensSortKey(group.scopeTokens)}`;

                    return (
                      <RuleGroupRow
                        key={groupKey}
                        $gap={6}
                        $active={active}
                        title="View this rule group"
                        {...templateRowActivationProps(openGroup)}
                      >
                        <FlexRow $gap={8} $alignItems="center" style={{ flexWrap: 'wrap' }}>
                          <Typography size={TypographySize.XS}>Group {group.groupIndex + 1}</Typography>
                          <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                            {group.templates.length}{' '}
                            {group.templates.length === 1 ? 'template' : 'templates'}
                          </Typography>
                        </FlexRow>
                        <ScopeTokens tokens={group.scopeTokens} />
                        {group.groupNotes?.trim() ? (
                          <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                            {group.groupNotes.trim()}
                          </Typography>
                        ) : null}
                      </RuleGroupRow>
                    );
                  })}
                </RuleGroupList>
              </DrawerCollapseSection>
            </>
          ) : null}

          {otherGroupTemplates.length > 0 || (activeRuleGroup && onEditGroupTemplates) ? (
            <>
              <Divider />
              <DrawerCollapseSection
                title="Other templates in this group"
                summary={siblingTemplatesSummary}
                defaultOpen
                titleAction={
                  onEditGroupTemplates && activeRuleGroup ? (
                    <IconButton
                      data-id="url-templating-detail-edit-group"
                      icon={EditIcon}
                      size={IconButtonSize.XS}
                      onClick={(event) => {
                        event.stopPropagation();
                        onEditGroupTemplates(activeRuleGroup);
                      }}
                    />
                  ) : undefined
                }
              >
                <GroupTemplateList $gap={2}>
                  {otherGroupTemplates.map((template) => {
                    const openTemplate = () => {
                      if (!activeRuleGroup) {
                        return;
                      }
                      onSelectTemplate(detailFromCrTemplate(activeRuleGroup, template));
                    };
                    return (
                      <GroupTemplateRow
                        key={template}
                        $gap={8}
                        title="View template"
                        {...templateRowActivationProps(openTemplate)}
                      >
                        <TemplatePath template={template} disabled={detail.actionDisabled} />
                      </GroupTemplateRow>
                    );
                  })}
                  {otherGroupTemplates.length === 0 ? (
                    <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                      No other templates in this group
                    </Typography>
                  ) : null}
                </GroupTemplateList>
              </DrawerCollapseSection>
            </>
          ) : null}
        </DrawerBody>
      ) : null}
    </Drawer>
    <WarningModal
      isOpen={confirmDeleteOpen}
      title="Delete this template?"
      description="This template will be removed from the rule group and take effect in the cluster immediately."
      onClose={() => {
        if (!deleting) {
          setConfirmDeleteOpen(false);
          setDeleteError(null);
        }
      }}
      denyButton={{
        label: 'Go Back',
        onClick: () => {
          setConfirmDeleteOpen(false);
          setDeleteError(null);
        },
        disabled: deleting,
      }}
      approveButton={{
        label: 'Delete One Template',
        onClick: () => {
          void handleDeleteTemplate();
        },
        loading: deleting,
        disabled: deleting,
      }}
    >
      {deleteError ? <Note status={StatusType.Error} message={deleteError} fullWidth /> : null}
    </WarningModal>
    </>
  );
}
