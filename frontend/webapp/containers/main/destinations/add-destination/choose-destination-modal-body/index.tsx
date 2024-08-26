import React, { useEffect, useState } from 'react';

import { SideMenu } from '@/components';
import { DestinationsList } from '../destinations-list';
import { Body, Container, SideMenuWrapper } from '../styled';
import { Divider, SectionTitle } from '@/reuseable-components';
import { DestinationFilterComponent } from '../choose-destination-menu';
import {
  DestinationsCategory,
  DestinationTypeItem,
  DropdownOption,
  GetDestinationTypesResponse,
  StepProps,
} from '@/types';
import { PotentialDestinationsList } from '../destinations-list/potential-destinations-list';
import { useQuery } from '@apollo/client';
import { GET_DESTINATION_TYPE } from '@/graphql';

interface ChooseDestinationModalBodyProps {
  data: DestinationTypeItem[];
  onSelect: (item: DestinationTypeItem) => void;
}

const SIDE_MENU_DATA: StepProps[] = [
  {
    title: 'DESTINATIONS',
    state: 'active',
    stepNumber: 1,
  },
  {
    title: 'CONNECTION',
    state: 'disabled',
    stepNumber: 2,
  },
];

const DEFAULT_MONITORS = ['logs', 'metrics', 'traces'];
const DEFAULT_DROPDOWN_VALUE = { id: 'all', value: 'All types' };

const CATEGORIES_DESCRIPTION = {
  managed: 'Effortless Monitoring with Scalable Performance Management',
  'self hosted':
    'Full Control and Customization for Advanced Application Monitoring',
};

export interface IDestinationListItem extends DestinationsCategory {
  description: string;
}

export function ChooseDestinationModalBody({
  onSelect,
}: ChooseDestinationModalBodyProps) {
  const { data } = useQuery<GetDestinationTypesResponse>(GET_DESTINATION_TYPE);
  const [searchValue, setSearchValue] = useState('');
  const [destinations, setDestinations] = useState<IDestinationListItem[]>([]);
  const [selectedMonitors, setSelectedMonitors] =
    useState<string[]>(DEFAULT_MONITORS);
  const [dropdownValue, setDropdownValue] = useState<DropdownOption>(
    DEFAULT_DROPDOWN_VALUE
  );

  useEffect(() => {
    if (data) {
      const destinationsCategories = data.destinationTypes.categories.map(
        (category) => {
          return {
            name: category.name,
            description: CATEGORIES_DESCRIPTION[category.name],
            items: category.items,
          };
        }
      );
      setDestinations(destinationsCategories);
    }
  }, [data]);

  function handleTagSelect(option: DropdownOption) {
    setDropdownValue(option);
  }

  // function filterData() {
  //   let filteredData = data;

  //   if (searchValue) {
  //     filteredData = filteredData.filter((item) =>
  //       item.displayName.toLowerCase().includes(searchValue.toLowerCase())
  //     );
  //   }

  //   if (dropdownValue.id !== 'all') {
  //     filteredData = filteredData.filter(
  //       (item) => item.category === dropdownValue.id
  //     );
  //   }

  //   if (selectedMonitors.length) {
  //     filteredData = filteredData.filter((item) =>
  //       selectedMonitors.some(
  //         (monitor) => item.supportedSignals[monitor].supported
  //       )
  //     );
  //   }

  //   return filteredData;
  // }

  function onMonitorSelect(monitor: string) {
    if (selectedMonitors.includes(monitor)) {
      setSelectedMonitors(selectedMonitors.filter((item) => item !== monitor));
    } else {
      setSelectedMonitors([...selectedMonitors, monitor]);
    }
  }

  return (
    <Container>
      <SideMenuWrapper>
        <SideMenu data={SIDE_MENU_DATA} currentStep={1} />
      </SideMenuWrapper>
      <Body>
        <SectionTitle
          title="Choose destination"
          description="Add backend destination you want to connect with Odigos."
        />
        <DestinationFilterComponent
          selectedTag={dropdownValue}
          onTagSelect={handleTagSelect}
          onSearch={setSearchValue}
          selectedMonitors={selectedMonitors}
          onMonitorSelect={onMonitorSelect}
        />
        <Divider margin="0 0 24px 0" />
        <DestinationsList items={destinations} setSelectedItems={onSelect} />
      </Body>
    </Container>
  );
}
