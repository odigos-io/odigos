import React from 'react';
import Image from 'next/image';
import { Text } from '../text';
import styled from 'styled-components';

type NoDataFoundProps = {
  title?: string;
  subTitle?: string;
};

const Title = styled(Text)`
  color: #7a7a7a;
  line-height: 24px;
`;

const SubTitle = styled(Text)`
  color: #525252;
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

const NoDataFound: React.FC<NoDataFoundProps> = ({ title = 'No data found', subTitle = 'Check your search phrase and try one more time' }) => {
  return (
    <Container>
      <TitleWrapper>
        <Image src='/icons/common/no-data-found.svg' alt='no-found' width={16} height={16} />
        <Title>{title}</Title>
      </TitleWrapper>
      {subTitle && <SubTitle>{subTitle}</SubTitle>}
    </Container>
  );
};

export { NoDataFound };
