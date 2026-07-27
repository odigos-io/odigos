'use client';

import React from 'react';
import styled, { type DefaultTheme } from 'styled-components';
import { FlexRow, Typography, TypographyColor, TypographySize } from '@odigos/ui-kit/components';
import type { ScopeToken } from './scope-token-types';

const chipTintByVariant: Record<ScopeToken['type'], (theme: DefaultTheme) => string> = {
  entire_cluster: (theme) => theme.v2.colors.purple[500],
  namespace: (theme) => theme.v2.colors.blue[500],
  language: (theme) => theme.v2.colors.green[500],
  workload: (theme) => theme.v2.colors.yellow[600],
};

const subjectColorByVariant: Record<ScopeToken['type'], (theme: DefaultTheme) => string> = {
  entire_cluster: (theme) => theme.v2.colors.purple[300],
  namespace: (theme) => theme.v2.colors.blue[400],
  language: (theme) => theme.v2.colors.green[600],
  workload: (theme) => theme.v2.colors.yellow[700],
};

const TokenChip = styled.span<{ $variant: ScopeToken['type'] }>`
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0;
  max-width: 100%;
  padding: 1px 6px;
  border-radius: 4px;
  border: 1px solid
    ${({ theme, $variant }) =>
      `color-mix(in srgb, ${chipTintByVariant[$variant](theme)} 18%, ${theme.v2.colors.silver[600]})`};
  background: ${({ theme, $variant }) =>
    `color-mix(in srgb, ${chipTintByVariant[$variant](theme)} 10%, ${theme.v2.colors.silver[800]})`};
  color: ${({ theme }) => theme.v2.colors.grey[400]};
  font-family: ${({ theme }) => theme.font_family.primary};
  font-size: ${({ theme }) => theme.v2.text.size.xxxs}px;
  line-height: 1.3;
  letter-spacing: 0.01em;
`;

const TokenSubject = styled.span<{ $variant: ScopeToken['type'] }>`
  font-weight: 500;
  color: ${({ theme, $variant }) => subjectColorByVariant[$variant](theme)};
`;

const TokenValue = styled.span`
  font-weight: 400;
  color: inherit;
`;

const WorkloadGroupSeparator = styled.span`
  display: inline-block;
  width: 6px;
`;

function formatListValues(values: string[]): string {
  return values.join(' / ');
}

function ScopeTokenChip({ token }: { token: ScopeToken }) {
  switch (token.type) {
    case 'entire_cluster':
      return (
        <TokenChip $variant={token.type}>
          <TokenSubject $variant="entire_cluster">Entire cluster</TokenSubject>
        </TokenChip>
      );
    case 'namespace':
      return (
        <TokenChip $variant={token.type}>
          <TokenSubject $variant="namespace">namespace:</TokenSubject>
          <TokenValue> {formatListValues(token.values)}</TokenValue>
        </TokenChip>
      );
    case 'language':
      return (
        <TokenChip $variant={token.type}>
          <TokenSubject $variant="language">language:</TokenSubject>
          <TokenValue> {formatListValues(token.values)}</TokenValue>
        </TokenChip>
      );
    case 'workload':
      return (
        <TokenChip $variant={token.type}>
          {token.groups.map((group, groupIndex) => (
            <React.Fragment key={`${group.kind}-${groupIndex}`}>
              {groupIndex > 0 && <WorkloadGroupSeparator />}
              <TokenSubject $variant="workload">{group.kind}:</TokenSubject>
              <TokenValue> {group.namespaceNames.join(', ')}</TokenValue>
            </React.Fragment>
          ))}
        </TokenChip>
      );
    default:
      return null;
  }
}

export function ScopeTokens({ tokens }: { tokens: ScopeToken[] }) {
  if (!tokens.length) {
    return null;
  }

  return (
    <FlexRow $gap={4} $alignItems="center" style={{ flexWrap: 'wrap' }}>
      {tokens.map((token, index) => (
        <React.Fragment key={`${token.type}-${index}-${scopeTokenReactKey(token)}`}>
          {index > 0 && (
            <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
              AND
            </Typography>
          )}
          <ScopeTokenChip token={token} />
        </React.Fragment>
      ))}
    </FlexRow>
  );
}

function scopeTokenReactKey(token: ScopeToken): string {
  switch (token.type) {
    case 'entire_cluster':
      return 'cluster';
    case 'namespace':
      return token.values.join(',');
    case 'language':
      return token.values.join(',');
    case 'workload':
      return token.groups.map((g) => `${g.kind}:${g.namespaceNames.join(',')}`).join('|');
    default:
      return '';
  }
}
