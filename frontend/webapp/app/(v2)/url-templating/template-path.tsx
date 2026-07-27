'use client';

import React, { useMemo } from 'react';
import styled, { type DefaultTheme, useTheme } from 'styled-components';

export type TemplateSegmentKind = 'static' | 'wildcard' | 'variable';

export type TemplatePathSegment = {
  kind: TemplateSegmentKind;
  text: string;
};

const VARIABLE_SEGMENT = /^\{[^{}]+\}$/;

export function parseTemplatePath(template: string): TemplatePathSegment[] {
  const trimmed = template.trim();
  if (!trimmed) {
    return [];
  }

  const segments: TemplatePathSegment[] = [];
  for (const part of trimmed.split('/')) {
    if (part === '') {
      continue;
    }
    if (part === '*') {
      segments.push({ kind: 'wildcard', text: part });
    } else if (VARIABLE_SEGMENT.test(part)) {
      segments.push({ kind: 'variable', text: part });
    } else {
      segments.push({ kind: 'static', text: part });
    }
  }
  return segments;
}

const segmentColor: Record<TemplateSegmentKind, (theme: DefaultTheme) => string> = {
  static: (theme) => theme.v2.colors.white[500],
  wildcard: (theme) => theme.v2.colors.orange[600],
  variable: (theme) => theme.v2.colors.blue[200],
};

const PathRoot = styled.span<{ $fontSize: number; $disabled?: boolean }>`
  display: inline;
  font-family: ${({ theme }) => theme.font_family.primary};
  font-size: ${({ $fontSize }) => $fontSize}px;
  line-height: 1.4;
  opacity: ${({ $disabled }) => ($disabled ? 0.45 : 1)};
  text-decoration: ${({ $disabled }) => ($disabled ? 'line-through' : 'none')};
  text-decoration-color: ${({ theme, $disabled }) => ($disabled ? theme.v2.colors.grey[500] : 'currentColor')};
`;

const Slash = styled.span`
  color: ${({ theme }) => theme.v2.colors.white[500]};
  font-weight: 400;
`;

const Segment = styled.span<{ $kind: TemplateSegmentKind }>`
  color: ${({ theme, $kind }) => segmentColor[$kind](theme)};
  font-weight: ${({ $kind }) => ($kind === 'wildcard' ? 700 : $kind === 'variable' ? 500 : 400)};
  font-style: ${({ $kind }) => ($kind === 'wildcard' ? 'italic' : 'normal')};
`;

export function templatePathFromSegments(segments: TemplatePathSegment[]): string {
  if (!segments.length) {
    return '';
  }
  return `/${segments.map((segment) => segment.text).join('/')}`;
}

export function TemplatePathSegmentLabel({
  kind,
  text,
  fontSize,
}: {
  kind: TemplateSegmentKind;
  text: string;
  fontSize?: number;
}) {
  const theme = useTheme();
  const size = fontSize ?? theme.v2.text.size.xs;
  return (
    <PathRoot $fontSize={size}>
      <Segment $kind={kind}>{text}</Segment>
    </PathRoot>
  );
}

type TemplatePathProps = {
  template: string;
  fontSize?: number;
  disabled?: boolean;
};

export function TemplatePath({ template, fontSize, disabled }: TemplatePathProps) {
  const theme = useTheme();
  const segments = useMemo(() => parseTemplatePath(template), [template]);
  const size = fontSize ?? theme.v2.text.size.xs;

  if (!segments.length) {
    return (
      <PathRoot $fontSize={size} $disabled={disabled} title={template}>
        {template}
      </PathRoot>
    );
  }

  const hasLeadingSlash = template.trimStart().startsWith('/');

  return (
    <PathRoot $fontSize={size} $disabled={disabled} title={template}>
      {hasLeadingSlash ? <Slash>/</Slash> : null}
      {segments.map((segment, index) => (
        <React.Fragment key={`${index}-${segment.text}`}>
          {index > 0 ? <Slash>/</Slash> : null}
          <Segment $kind={segment.kind}>{segment.text}</Segment>
        </React.Fragment>
      ))}
    </PathRoot>
  );
}
