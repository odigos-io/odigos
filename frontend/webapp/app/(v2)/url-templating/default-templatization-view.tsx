'use client';

import React, { useMemo, useState } from 'react';
import styled from 'styled-components';
import { ChevronRightIcon } from '@odigos/ui-kit/icons';
import { Badge, FlexColumn, FlexRow, Typography, TypographyColor, TypographySize } from '@odigos/ui-kit/components';
import { ActionOriginBadge } from './action-origin-badge';
import { ScopeTokens } from './scope-tokens';
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

function defaultGroupKey(group: DefaultTemplatizationCrGroup): string {
  return `${group.actionId}:${group.groupIndex}:${scopeTokensSortKey(group.scopeTokens)}`;
}

function DefaultRuleGroupCard({
  group,
  actionDisabled,
}: {
  group: DefaultTemplatizationCrGroup;
  actionDisabled: boolean;
}) {
  const [open, setOpen] = useState(true);
  const muted = group.disabled || actionDisabled;

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
          <span style={actionDisabled ? { opacity: 0.55 } : undefined}>
            <ScopeTokens tokens={group.scopeTokens} />
          </span>
          {group.disabled ? <Badge label="Disabled" /> : <Badge label="Enabled" />}
        </CrGroupHeaderMain>
      </CrGroupHeader>

      {open ? (
        <FlexColumn $gap={6}>
          <Typography
            size={TypographySize.XS}
            color={muted ? TypographyColor.Secondary : undefined}
          >
            {group.configLabel}
          </Typography>
          <FlexRow $gap={6} $alignItems="center" style={{ flexWrap: 'wrap' }}>
            {group.skipForNonSuccessCodes ? <Badge label="Skip non-2xx" /> : null}
            {group.skipHttpStatusCodes.length > 0 ? (
              <Badge label={`Skip HTTP ${group.skipHttpStatusCodes.join(', ')}`} />
            ) : null}
          </FlexRow>
        </FlexColumn>
      ) : null}
    </CrRuleGroupBlock>
  );
}

export function DefaultTemplatizationCrGroupsView({
  groups,
}: {
  groups: DefaultTemplatizationCrGroup[];
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
            />
          ))}
        </CrActionBlock>
      ))}
    </FlexColumn>
  );
}
