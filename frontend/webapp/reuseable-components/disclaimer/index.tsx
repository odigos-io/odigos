import React from 'react';
import styled from 'styled-components';
import { Text } from '../text';
import Image from 'next/image';
import theme from '@/styles/theme';

const Wrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 24px;
  padding: 12px 16px;
  border-radius: 32px;
  background: ${({ theme }) => theme.colors.card};
`;

interface Props {
  text: string;
  style?: React.CSSProperties;
}

const Disclaimer = ({ text, style }: Props) => {
  return (
    <Wrapper style={style}>
      <Image width={16} height={16} src='/icons/common/info.svg' alt='' />
      <Text size={14} color={theme.text.grey}>
        {text}
      </Text>
    </Wrapper>
  );
};

export default Disclaimer;
