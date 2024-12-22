import React from 'react';
import { Text } from '../text';
import { Tooltip } from '../tooltip';
import styled from 'styled-components';
import { OverviewIcon, SVG } from '@/assets';
import { hexPercentValues } from '@/styles/theme';

// Define types for the Tab component props
interface TabProps {
  title: string;
  tooltip?: string;
  icon: SVG;
  selected: boolean;
  disabled?: boolean;
  onClick?: () => void;
}

// Define types for the TabList component props
interface TabListProps {
  tabs?: TabProps[];
}

// Styled-components for Tab and TabList
const TabContainer = styled.div<{ $selected: TabProps['selected']; $disabled: TabProps['disabled']; $noClick: boolean }>`
  display: flex;
  align-items: center;
  padding: 10px 12px;
  border-radius: 32px;
  cursor: ${({ $noClick, $disabled }) => ($noClick ? 'unset' : $disabled ? 'not-allowed' : 'pointer')};
  background-color: ${({ $noClick, $selected, theme }) => ($noClick ? 'transparent' : $selected ? theme.colors.majestic_blue + hexPercentValues['024'] : theme.colors.card)};
  opacity: ${({ $disabled }) => ($disabled ? 0.5 : 1)};
  transition: background-color 0.3s, color 0.3s;

  &:hover {
    background-color: ${({ $noClick, $disabled, theme }) => ($noClick || $disabled ? 'none' : theme.colors.majestic_blue + hexPercentValues['024'])};
  }

  svg {
    margin-right: 8px;
  }
`;

const TabListContainer = styled.div`
  display: flex;
  gap: 8px;
`;

const Tab: React.FC<TabProps> = ({ title, tooltip, icon: Icon, selected, disabled, onClick }) => {
  return (
    <Tooltip text={tooltip}>
      <TabContainer $selected={selected} $disabled={disabled} $noClick={!onClick} onClick={onClick}>
        <Icon size={14} />
        <Text size={14}>{title}</Text>
      </TabContainer>
    </Tooltip>
  );
};

const TABS = [
  {
    title: 'Overview',
    icon: OverviewIcon,
    selected: true,
  },
  // {
  //   title: 'Service Map',
  //   icon: ServiceMapIcon,
  //   selected: false,
  //   onClick: () => {},
  //   disabled: true,
  //   tooltip: 'Coming soon',
  // },
  // {
  //   title: 'Trace View',
  //   icon: TraceViewIcon,
  //   selected: false,
  //   onClick: () => {},
  //   disabled: true,
  //   tooltip: 'Coming soon',
  // },
];

export const TabList: React.FC<TabListProps> = ({ tabs = TABS }) => {
  return (
    <TabListContainer>
      {tabs.map((tab) => (
        <Tab key={`tab-${tab.title}`} {...tab} />
      ))}
    </TabListContainer>
  );
};
