import React from 'react';
import styled from 'styled-components';
import { Theme } from '@odigos/ui-theme';
import { useDarkModeStore } from '@/store';
import { OdigosLogoText } from '@odigos/ui-icons';
import { FlexRow, NavigationButtons, type NavigationButtonsProps, Text, ToggleDarkMode } from '@odigos/ui-components';

interface Props extends NavigationButtonsProps {}

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

export const SetupHeader: React.FC<Props> = ({ buttons }) => {
  const { darkMode, setDarkMode } = useDarkModeStore();

  return (
    <Container>
      <OdigosLogoText size={80} />

      <Title family='secondary'>START WITH ODIGOS</Title>

      <FlexRow>
        <ToggleDarkMode darkMode={darkMode} setDarkMode={setDarkMode} />
        <NavigationButtons buttons={buttons} />
      </FlexRow>
    </Container>
  );
};
