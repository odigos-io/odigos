import React from 'react';
import styled from 'styled-components';
import { ToggleDarkMode } from '@/components/common';
import { FlexRow, OdigosLogoText, Theme } from '@odigos/ui-components';
import { NavigationButtonProps, NavigationButtons, Text } from '@/reuseable-components';

interface Props {
  navigationButtons: NavigationButtonProps[];
}

const Container = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 80px;
  padding: 0 24px 0 32px;
  background-color: ${({ theme }) => theme.colors.dark_grey};
  border-bottom: 1px solid ${({ theme }) => theme.colors.border + Theme.hexPercent['050']};
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

      <FlexRow>
        <ToggleDarkMode />
        <NavigationButtons buttons={navigationButtons} />
      </FlexRow>
    </Container>
  );
};
