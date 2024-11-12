import Image from 'next/image';
import React from 'react';
import { Text } from '../text';
import { Tooltip } from '../tooltip';
import styled from 'styled-components';
<<<<<<< HEAD
=======
import { hexPercentValues } from '@/styles/theme';
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866

// Define types for the Tab component props
interface TabProps {
  title: string;
  tooltip?: string;
  icon: string;
  selected: boolean;
  disabled?: boolean;
  onClick: () => void;
}

// Define types for the TabList component props
interface TabListProps {
  tabs?: TabProps[];
}

// Styled-components for Tab and TabList
const TabContainer = styled.div<{ selected: TabProps['selected']; disabled: TabProps['disabled'] }>`
  display: flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  padding: 0px 12px;
  border-radius: 32px;
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
<<<<<<< HEAD
  background-color: ${({ selected, theme }) => (selected ? theme.colors.majestic_blue + '3D' : theme.colors.card)};
=======
  background-color: ${({ selected, theme }) => (selected ? theme.colors.majestic_blue + hexPercentValues['024'] : theme.colors.card)};
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  opacity: ${({ disabled }) => (disabled ? 0.5 : 1)};
  transition: background-color 0.3s, color 0.3s;

  &:hover {
<<<<<<< HEAD
    background-color: ${({ disabled, theme }) => (disabled ? 'none' : theme.colors.majestic_blue + '3D')};
=======
    background-color: ${({ disabled, theme }) => (disabled ? 'none' : theme.colors.majestic_blue + hexPercentValues['024'])};
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  }

  svg {
    margin-right: 8px;
  }
`;

const TabListContainer = styled.div`
  display: flex;
  gap: 8px;
`;

// Tab component
const Tab: React.FC<TabProps> = ({ title, tooltip, icon, selected, disabled, onClick }) => {
  return (
    <Tooltip text={tooltip || ''}>
      <TabContainer selected={selected} disabled={disabled} onClick={onClick}>
        <Image src={icon} width={14} height={14} alt={title} />
        <Text size={14}>{title}</Text>
      </TabContainer>
    </Tooltip>
  );
};

const TABS = [
  {
    title: 'Overview',
    icon: '/icons/overview/overview.svg',
    selected: true,
    onClick: () => {},
  },
  {
    title: 'Service map',
    icon: '/icons/overview/service-map.svg',
    selected: false,
    onClick: () => {},
    disabled: true,
    tooltip: 'Coming soon',
  },
  {
    title: 'Trace view',
    icon: '/icons/overview/trace-view.svg',
    selected: false,
    onClick: () => {},
    disabled: true,
    tooltip: 'Coming soon',
  },
];

const TabList: React.FC<TabListProps> = ({ tabs = TABS }) => {
  return (
    <TabListContainer>
      {tabs.map((tab) => (
        <Tab key={`tab-${tab.title}`} {...tab} />
      ))}
    </TabListContainer>
  );
};

export { TabList };
