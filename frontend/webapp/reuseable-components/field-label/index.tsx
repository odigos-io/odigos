import React from 'react';
import { Text } from '../text';
import { Tooltip } from '../tooltip';
import styled from 'styled-components';

interface Props {
  title?: string;
  required?: boolean;
  tooltip?: string;
  style?: React.CSSProperties;
}

const Wrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
`;

const Title = styled(Text)`
  font-size: 14px;
  opacity: 0.8;
  line-height: 22px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
`;

const OptionalText = styled(Text)`
  font-size: 12px;
  color: #7a7a7a;
  opacity: 0.8;
`;

const FieldLabel: React.FC<Props> = ({ title, required, tooltip, style }) => {
  if (!title) return null;

  return (
    <Wrapper style={style}>
      <Tooltip text={tooltip} withIcon>
        <Title>{title}</Title>
        {!required && <OptionalText>(optional)</OptionalText>}
      </Tooltip>
    </Wrapper>
  );
};

export { FieldLabel };
