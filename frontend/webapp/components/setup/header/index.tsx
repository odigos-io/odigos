import React from 'react';
import styled from 'styled-components';
import { OdigosLogoText } from '@/assets';
import { NavigationButtonProps, NavigationButtons, Text } from '@/reuseable-components';

interface Props {
  navigationButtons: NavigationButtonProps[];
}

const Container = styled.div`
  display: flex;
  justify-content: space-between;
  padding: 0 24px 0 32px;
  align-items: center;
  background-color: ${({ theme }) => theme.colors.dark_grey};
  border-bottom: 1px solid rgba(249, 249, 249, 0.16);
  height: 80px;
`;

const Title = styled(Text)`
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
`;

export const SetupHeader: React.FC<Props> = ({ navigationButtons }) => {
  return (
    <Container>
      <OdigosLogoText size={80} />
      <Title family='secondary'>START WITH ODIGOS</Title>
      <NavigationButtons buttons={navigationButtons} />
    </Container>
  );
};
