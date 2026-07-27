'use client';

import React, { useMemo, useState } from 'react';
import styled, { css } from 'styled-components';
import { ChevronRightIcon, EditIcon } from '@odigos/ui-kit/icons';
import { Badge, FlexColumn, FlexRow, IconButton, IconButtonSize, Typography, TypographyColor, TypographySize } from '@odigos/ui-kit/components';
import { ActionOriginBadge } from './action-origin-badge';
import { buildCustomTemplateTree, type CustomTemplateTreeNode } from './custom-template-tree';
import { ScopeTokens } from './scope-tokens';
import { TemplatePath, type TemplatePathSegment, templatePathFromSegments } from './template-path';
import type { AggregatedCustomTemplate, CustomTemplateCrGroup } from './url-templatization-aggregate';
import { scopeTokensSortKey } from './url-templatization-aggregate';
import { detailFromAggregatedTemplate, detailFromCrTemplate, type UrlTemplateDetail } from './template-detail-types';
import { stopRowActivation, templateRowActivationProps } from './template-row-activation';

export type EditGroupTemplatesOptions = {
  focusNewTemplate?: boolean;
};

export type OnSelectUrlTemplate = (detail: UrlTemplateDetail) => void;

const interactiveTemplateRow = css`
  cursor: pointer;
  border-radius: 6px;
  margin-left: -6px;
  margin-right: -6px;
  padding-left: 6px;
  padding-right: 6px;
  transition: background 0.12s ease;

  &:hover {
    background: ${({ theme }) => theme.v2.colors.silver[800]};
  }

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.v2.colors.blue[400]};
    outline-offset: 2px;
  }
`;

const TreeRow = styled(FlexRow)<{ $depth: number; $interactive?: boolean }>`
  width: 100%;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid ${({ theme }) => theme.v2.colors.silver[600]};
  user-select: none;
  ${({ $interactive }) => ($interactive ? interactiveTemplateRow : '')}
  /* Keep depth indent after interactive styles (those set padding-left: 6px). */
  padding-left: ${({ $depth, $interactive }) => $depth * 16 + ($interactive ? 6 : 0)}px;

  &:last-child {
    border-bottom: none;
  }
`;

const TreeRowLead = styled(FlexRow)<{ $clickable?: boolean }>`
  flex-shrink: 0;
  align-items: center;
  width: 14px;
  justify-content: center;
  cursor: ${({ $clickable }) => ($clickable ? 'pointer' : 'default')};
`;

const TreeChevron = styled.span<{ $open: boolean }>`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transform: rotate(${({ $open }) => ($open ? '90deg' : '0deg')});
  transition: transform 0.15s ease;
  color: ${({ theme }) => theme.v2.colors.grey[400]};
`;

const TreePathAndScope = styled(FlexRow)`
  flex: 1;
  min-width: 0;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
`;

const ListRow = styled(FlexRow)<{ $interactive?: boolean }>`
  width: 100%;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid ${({ theme }) => theme.v2.colors.silver[600]};
  ${({ $interactive }) => ($interactive ? interactiveTemplateRow : '')}

  &:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }
`;

const ListRowMain = styled(FlexColumn)`
  flex: 1;
  min-width: 0;
  gap: 4px;
`;

const ListRowTitleLine = styled(FlexRow)`
  flex: 1;
  min-width: 0;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
`;

function customTemplateRowKey(item: AggregatedCustomTemplate): string {
  return `${item.actionId}|${item.template}|${scopeTokensSortKey(item.scopeTokens)}`;
}

export function CustomTemplateListRow({
  item,
  onSelectTemplate,
}: {
  item: AggregatedCustomTemplate;
  onSelectTemplate: OnSelectUrlTemplate;
}) {
  const openDetail = () => onSelectTemplate(detailFromAggregatedTemplate(item));

  return (
    <ListRow
      $gap={12}
      $interactive
      title="View template details"
      {...templateRowActivationProps(openDetail)}
    >
      <ListRowMain $gap={4}>
        <ListRowTitleLine $gap={8}>
          <TemplatePath template={item.template} disabled={item.actionDisabled} />
          <span style={item.actionDisabled ? { opacity: 0.55 } : undefined}>
            <ScopeTokens tokens={item.scopeTokens} />
          </span>
        </ListRowTitleLine>
        <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
          In {item.actionLabels.join(', ')}
        </Typography>
      </ListRowMain>
      <FlexRow $gap={6} $alignItems="center">
        <ActionOriginBadge uiGenerated={item.actionUiGenerated} />
        {item.actionDisabled ? <Badge label="Disabled" /> : null}
        {item.count > 1 ? <Badge label={`×${item.count}`} /> : null}
      </FlexRow>
    </ListRow>
  );
}

function TreeNodeRow({
  depth,
  fullPath,
  entries,
  expandable,
  open,
  onToggle,
  onSelectTemplate,
}: {
  depth: number;
  fullPath: string;
  entries: AggregatedCustomTemplate[];
  expandable: boolean;
  open: boolean;
  onToggle: () => void;
  onSelectTemplate: OnSelectUrlTemplate;
}) {
  const actionsHint = entries
    .flatMap((entry) => entry.actionLabels)
    .filter((label, index, labels) => labels.indexOf(label) === index)
    .join(', ');

  const pathDisabled = entries.length > 0 && entries.every((entry) => entry.actionDisabled);

  return (
    <>
      {entries.map((entry, entryIndex) => {
        const openDetail = () => onSelectTemplate(detailFromAggregatedTemplate(entry));
        const showChevron = entryIndex === 0 && expandable;

        return (
          <TreeRow
            key={customTemplateRowKey(entry)}
            $depth={depth}
            $gap={12}
            $interactive
            title={actionsHint ? `In ${actionsHint}` : 'View template details'}
            {...templateRowActivationProps(openDetail)}
          >
            <TreeRowLead
              $clickable={showChevron}
              onClick={
                showChevron
                  ? (event) => {
                      stopRowActivation(event);
                      onToggle();
                    }
                  : undefined
              }
              title={showChevron ? (open ? 'Collapse' : 'Expand') : undefined}
            >
              {showChevron ? (
                <TreeChevron $open={open}>
                  <ChevronRightIcon size={12} />
                </TreeChevron>
              ) : null}
            </TreeRowLead>
            <TreePathAndScope $gap={8}>
              <TemplatePath template={fullPath} disabled={entry.actionDisabled || pathDisabled} />
              <span style={entry.actionDisabled ? { opacity: 0.55 } : undefined}>
                <ScopeTokens tokens={entry.scopeTokens} />
              </span>
              {entry.count > 1 ? <Badge label={`×${entry.count}`} /> : null}
            </TreePathAndScope>
          </TreeRow>
        );
      })}
    </>
  );
}

function TreeBranch({
  node,
  ancestorSegments,
  depth,
  onSelectTemplate,
}: {
  node: CustomTemplateTreeNode;
  ancestorSegments: TemplatePathSegment[];
  depth: number;
  onSelectTemplate: OnSelectUrlTemplate;
}) {
  const [open, setOpen] = useState(true);
  const nodeSegment: TemplatePathSegment = { kind: node.kind, text: node.segment };
  const pathSegments = [...ancestorSegments, nodeSegment];
  const fullPath = templatePathFromSegments(pathSegments);
  const hasChildren = node.children.length > 0;
  const hasEntries = node.entries.length > 0;

  if (!hasEntries) {
    if (!hasChildren) {
      return null;
    }
    return (
      <>
        {node.children.map((child, index) => (
          <TreeBranch
            key={`${fullPath}/${child.kind}:${child.segment}:${index}`}
            node={child}
            ancestorSegments={pathSegments}
            depth={depth}
            onSelectTemplate={onSelectTemplate}
          />
        ))}
      </>
    );
  }

  return (
    <>
      <TreeNodeRow
        depth={depth}
        fullPath={fullPath}
        entries={node.entries}
        expandable={hasChildren}
        open={open}
        onToggle={() => setOpen((value) => !value)}
        onSelectTemplate={onSelectTemplate}
      />
      {hasChildren && open
        ? node.children.map((child, index) => (
            <TreeBranch
              key={`${fullPath}/${child.kind}:${child.segment}:${index}`}
              node={child}
              ancestorSegments={pathSegments}
              depth={depth + 1}
              onSelectTemplate={onSelectTemplate}
            />
          ))
        : null}
    </>
  );
}

export function CustomTemplateTreeView({
  items,
  onSelectTemplate,
}: {
  items: AggregatedCustomTemplate[];
  onSelectTemplate: OnSelectUrlTemplate;
}) {
  const roots = useMemo(() => buildCustomTemplateTree(items), [items]);

  if (!roots.length) {
    return null;
  }

  return (
    <FlexColumn $gap={0} style={{ width: '100%' }}>
      {roots.map((node, index) => (
        <TreeBranch
          key={`${node.kind}:${node.segment}:${index}`}
          node={node}
          ancestorSegments={[]}
          depth={0}
          onSelectTemplate={onSelectTemplate}
        />
      ))}
    </FlexColumn>
  );
}

export function CustomTemplateListView({
  items,
  onSelectTemplate,
}: {
  items: AggregatedCustomTemplate[];
  onSelectTemplate: OnSelectUrlTemplate;
}) {
  return (
    <>
      {items.map((item) => (
        <CustomTemplateListRow key={customTemplateRowKey(item)} item={item} onSelectTemplate={onSelectTemplate} />
      ))}
    </>
  );
}

const CrActionBlock = styled(FlexColumn)`
  width: 100%;
  gap: 12px;
  padding: 16px 0;
  border-bottom: 2px solid ${({ theme }) => theme.v2.colors.silver[500]};

  &:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }
`;

const CrRuleGroupBlock = styled(FlexColumn)`
  width: 100%;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 8px;
  background: ${({ theme }) => theme.v2.colors.silver[900]};
  border: 1px solid ${({ theme }) => theme.v2.colors.silver[700]};
`;

const CrGroupHeader = styled(FlexRow)`
  width: 100%;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
  border-radius: 6px;
  margin: -4px -6px;
  padding: 4px 6px;

  &:hover {
    background: ${({ theme }) => theme.v2.colors.silver[800]};
  }

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.v2.colors.blue[400]};
    outline-offset: 2px;
  }
`;

const CrGroupChevron = styled.span<{ $open: boolean }>`
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  transform: rotate(${({ $open }) => ($open ? '90deg' : '0deg')});
  transition: transform 0.15s ease;
  color: ${({ theme }) => theme.v2.colors.grey[400]};
`;

const CrGroupHeaderMain = styled(FlexRow)`
  flex: 1;
  min-width: 0;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
`;

const CrTemplateLine = styled(FlexRow)<{ $interactive?: boolean }>`
  width: 100%;
  align-items: center;
  padding: 6px 8px;
  margin-left: -8px;
  border-left: 2px solid ${({ theme }) => theme.v2.colors.silver[600]};
  ${({ $interactive }) => ($interactive ? interactiveTemplateRow : '')}
`;

const AddTemplatesLink = styled.button`
  align-self: flex-start;
  margin-top: 4px;
  padding: 4px 0;
  border: none;
  background: none;
  cursor: pointer;
  font-family: ${({ theme }) => theme.font_family.primary};
  font-size: ${({ theme }) => theme.v2.text.size.xxxs}px;
  line-height: 1.4;
  color: ${({ theme }) => theme.v2.colors.blue[400]};

  &:hover {
    text-decoration: underline;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    text-decoration: none;
  }
`;

const AddGroupButton = styled.button`
  flex-shrink: 0;
  padding: 3px 8px;
  border-radius: 6px;
  border: 1px solid ${({ theme }) => theme.v2.colors.silver[600]};
  background: ${({ theme }) => theme.v2.colors.silver[800]};
  cursor: pointer;
  font-family: ${({ theme }) => theme.font_family.primary};
  font-size: ${({ theme }) => theme.v2.text.size.xxxs}px;
  line-height: 1.3;
  color: ${({ theme }) => theme.v2.colors.white[500]};

  &:hover {
    background: ${({ theme }) => theme.v2.colors.silver[700]};
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
`;

function crGroupKey(group: CustomTemplateCrGroup): string {
  return `${group.actionId}:${group.groupIndex}:${scopeTokensSortKey(group.scopeTokens)}`;
}

function CrRuleGroupCard({
  group,
  actionDisabled,
  onSelectTemplate,
  onEditGroupTemplates,
}: {
  group: CustomTemplateCrGroup;
  actionDisabled: boolean;
  onSelectTemplate: OnSelectUrlTemplate;
  onEditGroupTemplates?: (group: CustomTemplateCrGroup, options?: EditGroupTemplatesOptions) => void;
}) {
  const [open, setOpen] = useState(true);
  const groupKey = crGroupKey(group);

  const toggleOpen = () => setOpen((value) => !value);

  return (
    <CrRuleGroupBlock $gap={8}>
      <CrGroupHeader
        $gap={8}
        role="button"
        tabIndex={0}
        aria-expanded={open}
        title={open ? 'Collapse rule group' : 'Expand rule group'}
        onClick={toggleOpen}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            toggleOpen();
          }
        }}
      >
        <CrGroupChevron $open={open}>
          <ChevronRightIcon size={12} />
        </CrGroupChevron>
        <CrGroupHeaderMain $gap={8}>
          <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
            Group {group.groupIndex + 1}
          </Typography>
          {onEditGroupTemplates ? (
            <IconButton
              data-id={`url-templating-edit-group-${groupKey}`}
              icon={EditIcon}
              size={IconButtonSize.XS}
              onClick={(event) => {
                stopRowActivation(event);
                onEditGroupTemplates(group);
              }}
            />
          ) : null}
          <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
            {group.templates.length} {group.templates.length === 1 ? 'template' : 'templates'}
          </Typography>
          <span style={actionDisabled ? { opacity: 0.55 } : undefined}>
            <ScopeTokens tokens={group.scopeTokens} />
          </span>
        </CrGroupHeaderMain>
      </CrGroupHeader>

      {open ? (
        <>
          {group.groupNotes?.trim() ? (
            <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
              {group.groupNotes.trim()}
            </Typography>
          ) : null}
          <FlexColumn $gap={4}>
            {group.templates.map((template) => {
              const openDetail = () => onSelectTemplate(detailFromCrTemplate(group, template));
              const rowActivation = templateRowActivationProps(openDetail);
              return (
                <CrTemplateLine
                  key={`${groupKey}:${template}`}
                  $gap={8}
                  $interactive
                  title="View template details"
                  {...rowActivation}
                  onClick={(event) => {
                    stopRowActivation(event);
                    rowActivation.onClick();
                  }}
                >
                  <TemplatePath template={template} disabled={actionDisabled} />
                </CrTemplateLine>
              );
            })}
            {group.templates.length === 0 ? (
              <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                No templates configured
              </Typography>
            ) : null}
          </FlexColumn>
          {onEditGroupTemplates ? (
            <AddTemplatesLink
              type="button"
              data-id={`url-templating-add-templates-${groupKey}`}
              onClick={(event) => {
                stopRowActivation(event);
                onEditGroupTemplates(group, { focusNewTemplate: true });
              }}
            >
              + Add New Templates
            </AddTemplatesLink>
          ) : null}
        </>
      ) : null}
    </CrRuleGroupBlock>
  );
}

export function CustomTemplateCrGroupsView({
  groups,
  onSelectTemplate,
  onEditGroupTemplates,
  onCreateGroup,
}: {
  groups: CustomTemplateCrGroup[];
  onSelectTemplate: OnSelectUrlTemplate;
  onEditGroupTemplates?: (group: CustomTemplateCrGroup, options?: EditGroupTemplatesOptions) => void;
  onCreateGroup?: (actionId: string) => void;
}) {
  const actionSections = useMemo(() => {
    const order: string[] = [];
    const byAction = new Map<
      string,
      {
        actionLabel: string;
        actionDisabled: boolean;
        actionUiGenerated: boolean;
        groups: CustomTemplateCrGroup[];
      }
    >();

    for (const group of groups) {
      if (!byAction.has(group.actionId)) {
        order.push(group.actionId);
        byAction.set(group.actionId, {
          actionLabel: group.actionLabel,
          actionDisabled: group.actionDisabled,
          actionUiGenerated: group.actionUiGenerated,
          groups: [],
        });
      }
      byAction.get(group.actionId)?.groups.push(group);
    }

    return order.map((actionId) => ({
      actionId,
      ...byAction.get(actionId)!,
    }));
  }, [groups]);

  return (
    <FlexColumn $gap={0} style={{ width: '100%' }}>
      {actionSections.map(({ actionId, actionLabel, actionDisabled, actionUiGenerated, groups: actionGroups }) => (
        <CrActionBlock key={actionId} $gap={12}>
          <FlexRow $gap={8} $alignItems="center" style={{ flexWrap: 'wrap' }}>
            <Typography size={TypographySize.XS}>{actionLabel}</Typography>
            <ActionOriginBadge uiGenerated={actionUiGenerated} />
            {actionDisabled ? <Badge label="Disabled" /> : null}
            <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
              {actionGroups.length} {actionGroups.length === 1 ? 'rule group' : 'rule groups'}
            </Typography>
            {onCreateGroup ? (
              <AddGroupButton
                type="button"
                data-id={`url-templating-create-group-${actionId}`}
                onClick={(event) => {
                  stopRowActivation(event);
                  onCreateGroup(actionId);
                }}
              >
                + Add group
              </AddGroupButton>
            ) : null}
          </FlexRow>
          {actionGroups.map((group) => (
            <CrRuleGroupCard
              key={crGroupKey(group)}
              group={group}
              actionDisabled={actionDisabled}
              onSelectTemplate={onSelectTemplate}
              onEditGroupTemplates={onEditGroupTemplates}
            />
          ))}
        </CrActionBlock>
      ))}
    </FlexColumn>
  );
}
