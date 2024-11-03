import React from 'react';
import Image from 'next/image';
import { Text } from '../text';
import { Tooltip } from '../tooltip';
import styled from 'styled-components';

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

const FieldLabel = ({ title, required, tooltip }: { title?: string; required?: boolean; tooltip?: string }) => {
  if (!title) return null;

  return (
    <Wrapper>
      <Title>{title}</Title>
      {!required && <OptionalText>(optional)</OptionalText>}
      {tooltip && (
        <Tooltip text={tooltip || ''}>
          <Image src='/icons/common/info.svg' alt='' width={16} height={16} style={{ marginBottom: 4 }} />
        </Tooltip>
      )}
    </Wrapper>
  );
};

export { FieldLabel };
