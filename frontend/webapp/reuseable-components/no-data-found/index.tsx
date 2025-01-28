import React from 'react';
import { Text } from '../text';
import { NoDataIcon } from '@/assets';
import styled, { useTheme } from 'styled-components';

interface Props {
  title?: string;
  subTitle?: string;
}

const Title = styled(Text)`
  color: ${({ theme }) => theme.text.darker_grey};
  line-height: 24px;
`;

const SubTitle = styled(Text)`
  color: ${({ theme }) => theme.colors.border};
  font-size: 14px;
  font-weight: 200;
  line-height: 20px;
`;

const TitleWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const Container = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
`;

export const NoDataFound: React.FC<Props> = ({ title = 'No data found', subTitle = 'Check your search phrase and try one more time' }) => {
  const theme = useTheme();

  return (
    <Container>
      <TitleWrapper>
        <NoDataIcon fill={theme.text.dark_grey} />
        <Title>{title}</Title>
      </TitleWrapper>
      {subTitle && <SubTitle>{subTitle}</SubTitle>}
    </Container>
  );
};
