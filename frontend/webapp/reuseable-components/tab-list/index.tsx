import Image from 'next/image';
import React from 'react';
import styled from 'styled-components';
import { Text } from '../text';

// Define types for the Tab component props
interface TabProps {
  title: string;
  icon: string;
  selected: boolean;
  onClick: () => void;
}

// Define types for the TabList component props
interface TabListProps {
  tabs?: TabProps[];
}

// Styled-components for Tab and TabList
const TabContainer = styled.div<{ selected: boolean }>`
  display: flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  padding: 0px 12px;
  border-radius: 32px;
  cursor: pointer;
  background-color: ${({ selected, theme }) =>
    selected ? theme.colors.selected_hover : theme.colors.card};
  transition: background-color 0.3s, color 0.3s;

  &:hover {
    background-color: ${({ theme }) => theme.colors.selected_hover};
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
const Tab: React.FC<TabProps> = ({ title, icon, selected, onClick }) => {
  return (
    <TabContainer selected={selected} onClick={onClick}>
      <Image src={icon} width={14} height={14} alt={title} />
      <Text size={14}>{title}</Text>
    </TabContainer>
  );
};

const TABS = [
  {
    title: 'Overview',
    icon: '/icons/overview/overview.svg',
    selected: true,
    onClick: () => {},
  },
];

// TabList component
const TabList: React.FC<TabListProps> = ({ tabs = TABS }) => {
  return (
    <TabListContainer>
      {tabs.map((tab, index) => (
        <Tab
          key={index}
          title={tab.title}
          icon={tab.icon}
          selected={tab.selected}
          onClick={tab.onClick}
        />
      ))}
    </TabListContainer>
  );
};

export { TabList };
