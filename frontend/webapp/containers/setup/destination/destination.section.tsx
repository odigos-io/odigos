import React, { useState } from 'react';
import { useQuery } from 'react-query';
import { useSectionData } from '@/hooks';
import { KeyvalLoader } from '@/design.system';
import { EmptyList } from '@/components/lists';
import { OVERVIEW, QUERIES, SETUP } from '@/utils';
import { getDestinationsTypes } from '@/services';
import { MONITORING_OPTIONS } from '@/utils/constants/monitors';
import { DestinationList, DestinationOptionMenu } from '@/components/setup';
import {
  LoaderWrapper,
  DestinationContainerWrapper,
} from './destination.section.styled';
import {
  sortDestinationList,
  filterDataByTextQuery,
  isDestinationListEmpty,
  filterDataByMonitorsOption,
} from './utils';

interface DestinationTypes {
  image_url: string;
  display_name: string;
  supported_signals: {
    [key: string]: {
      supported: boolean;
    };
  };
  type: string;
}

type DestinationSectionProps = {
  onSelectItem?: (type: string) => void;
};

export function DestinationSection({ onSelectItem }: DestinationSectionProps) {
  const [searchFilter, setSearchFilter] = useState<string>('');
  const [dropdownData, setDropdownData] = useState<any>(null);

  const { sectionData, setSectionData } = useSectionData({});

  const [monitoringOption, setMonitoringOption] =
    useState<any>(MONITORING_OPTIONS);

  const { isLoading, data } = useQuery(
    [QUERIES.API_DESTINATION_TYPES],
    getDestinationsTypes
  );

  function handleSelectItem(item: DestinationTypes) {
    setSectionData(item);
    onSelectItem && onSelectItem(item.type);
  }

  function renderDestinationLists() {
    sortDestinationList(data);
    const list = filterDataByMonitorsOption(
      filterDataByTextQuery(data, searchFilter),
      monitoringOption
    );

    if (isDestinationListEmpty(list, dropdownData?.id)) {
      return <EmptyList title={OVERVIEW.EMPTY_DESTINATION} />;
    }

    return list?.map((category: any) => {
      const displayItem =
        dropdownData?.label === category.name ||
        dropdownData?.label === SETUP.ALL;

      return (
        displayItem && (
          <DestinationList
            sectionData={sectionData}
            key={category.name}
            data={category}
            onItemClick={(item: DestinationTypes) => handleSelectItem(item)}
          />
        )
      );
    });
  }

  if (isLoading) {
    return (
      <LoaderWrapper>
        <KeyvalLoader />
      </LoaderWrapper>
    );
  }

  return (
    <DestinationContainerWrapper>
      <DestinationOptionMenu
        searchFilter={searchFilter}
        setSearchFilter={setSearchFilter}
        setDropdownData={setDropdownData}
        monitoringOption={monitoringOption}
        setMonitoringOption={setMonitoringOption}
        data={data?.categories}
      />
      {data && renderDestinationLists()}
    </DestinationContainerWrapper>
  );
}
