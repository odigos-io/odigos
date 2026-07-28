'use client';

import React, { useMemo } from 'react';
import styled from 'styled-components';
import { EditIcon } from '@odigos/ui-kit/icons';
import {
  Badge,
  FlexColumn,
  FlexRow,
  IconButton,
  IconButtonSize,
  Typography,
  TypographyColor,
  TypographySize,
} from '@odigos/ui-kit/components';
import { ActionOriginBadge } from './action-origin-badge';
import { ScopeTokens } from './scope-tokens';
import { stopRowActivation, templateRowActivationProps } from './template-row-activation';
import type { DefaultTemplatizationCrGroup } from './url-templatization-aggregate';
import { scopeTokensSortKey } from './url-templatization-aggregate';

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

const CrRuleGroupRow = styled(FlexRow)`
  width: 100%;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 8px;
  background: ${({ theme }) => theme.v2.colors.silver[900]};
  border: 1px solid ${({ theme }) => theme.v2.colors.silver[700]};
  flex-wrap: wrap;
  cursor: pointer;
  transition: background 0.12s ease, border-color 0.12s ease;

  &:hover {
    background: ${({ theme }) => theme.v2.colors.silver[800]};
    border-color: ${({ theme }) => theme.v2.colors.silver[600]};
  }

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.v2.colors.blue[400]};
    outline-offset: 2px;
  }
`;

function defaultGroupKey(group: DefaultTemplatizationCrGroup): string {
  return `${group.actionId}:${group.groupIndex}:${scopeTokensSortKey(group.scopeTokens)}`;
}

function DefaultRuleGroupCard({
  group,
  actionDisabled,
  onSelect,
  onEdit,
}: {
  group: DefaultTemplatizationCrGroup;
  actionDisabled: boolean;
  onSelect: (group: DefaultTemplatizationCrGroup) => void;
  onEdit?: (group: DefaultTemplatizationCrGroup) => void;
}) {
  const muted = group.disabled || actionDisabled;
  const groupKey = defaultGroupKey(group);
  const activation = templateRowActivationProps(() => onSelect(group));

  return (
    <CrRuleGroupRow $gap={8} title="View default templatization rule" {...activation}>
      {onEdit ? (
        <IconButton
          data-id={`url-templating-edit-default-${groupKey}`}
          icon={EditIcon}
          size={IconButtonSize.XS}
          onClick={(event) => {
            stopRowActivation(event);
            onEdit(group);
          }}
        />
      ) : null}
      <Typography size={TypographySize.XS} color={muted ? TypographyColor.Secondary : undefined}>
        {group.configLabel}
      </Typography>
      <span style={actionDisabled ? { opacity: 0.55 } : undefined}>
        <ScopeTokens tokens={group.scopeTokens} />
      </span>
      {group.disabled ? <Badge label="Default Templating Disabled" /> : <Badge label="Default Templating Enabled" />}
      {group.skipForNonSuccessCodes ? <Badge label="Skip non-2xx" /> : null}
      {group.skipHttpStatusCodes.length > 0 ? (
        <Badge label={`Skip HTTP ${group.skipHttpStatusCodes.join(', ')}`} />
      ) : null}
    </CrRuleGroupRow>
  );
}

export function DefaultTemplatizationCrGroupsView({
  groups,
  onSelectGroup,
  onEditGroup,
}: {
  groups: DefaultTemplatizationCrGroup[];
  onSelectGroup: (group: DefaultTemplatizationCrGroup) => void;
  onEditGroup?: (group: DefaultTemplatizationCrGroup) => void;
}) {
  const actionSections = useMemo(() => {
    const order: string[] = [];
    const byAction = new Map<
      string,
      {
        actionLabel: string;
        actionDisabled: boolean;
        actionUiGenerated: boolean;
        groups: DefaultTemplatizationCrGroup[];
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
          </FlexRow>
          {actionGroups.map((group) => (
            <DefaultRuleGroupCard
              key={defaultGroupKey(group)}
              group={group}
              actionDisabled={actionDisabled}
              onSelect={onSelectGroup}
              onEdit={onEditGroup}
            />
          ))}
        </CrActionBlock>
      ))}
    </FlexColumn>
  );
}
