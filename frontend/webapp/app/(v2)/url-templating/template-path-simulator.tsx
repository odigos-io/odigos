'use client';

import React, { useMemo, useState } from 'react';
import styled from 'styled-components';
import { Input, FlexColumn, Typography, TypographyColor, TypographySize } from '@odigos/ui-kit/components';
import { TemplatePath } from './template-path';
import { simulateCustomTemplate } from './template-simulator';

const SimulatorBlock = styled(FlexColumn)<{ $embedded?: boolean }>`
  width: 100%;
  gap: 8px;
  padding-top: ${({ $embedded }) => ($embedded ? 0 : '12px')};
  margin-top: ${({ $embedded }) => ($embedded ? 0 : '4px')};
  border-top: ${({ theme, $embedded }) => ($embedded ? 'none' : `1px solid ${theme.v2.colors.silver[700]}`)};
`;

type TemplatePathSimulatorProps = {
  template: string;
  actionDisabled?: boolean;
  embedded?: boolean;
};

export function TemplatePathSimulator({ template, actionDisabled, embedded }: TemplatePathSimulatorProps) {
  const [urlInput, setUrlInput] = useState('');

  const result = useMemo(
    () => simulateCustomTemplate(template, urlInput),
    [template, urlInput],
  );

  if (!template.trim()) {
    return null;
  }

  return (
    <SimulatorBlock $gap={8} $embedded={embedded}>
      <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
        Try a URL
      </Typography>
      <Input
        data-id="url-template-path-simulator-input"
        value={urlInput}
        onChange={(event) => setUrlInput(event.target.value)}
        placeholder="Paste a URL or path, e.g. https://api.example.com/users/42"
        width="100%"
      />
      {result.status === 'matched' ? (
        <FlexColumn $gap={4}>
          <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
            Templated path
          </Typography>
          <TemplatePath template={result.outputPath} disabled={actionDisabled} fontSize={12} />
        </FlexColumn>
      ) : null}
      {result.status === 'no_match' && urlInput.trim() ? (
        <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
          {result.message}
        </Typography>
      ) : null}
    </SimulatorBlock>
  );
}
