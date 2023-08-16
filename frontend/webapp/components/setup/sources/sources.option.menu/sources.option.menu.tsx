import React from 'react';
import { SourcesOptionMenuWrapper } from './sources.option.menu.styled';
import { FilterSourcesOptions } from './filter.sources.options';
import { ActionSourcesOptions } from './action.sources.options';

export function SourcesOptionMenu({
  setCurrentItem,
  data,
  searchFilter,
  setSearchFilter,
  onSelectAllChange,
  selectedApplications,
  currentNamespace,
  onFutureApplyChange,
}: any) {
  return (
    <SourcesOptionMenuWrapper>
      <FilterSourcesOptions
        setCurrentItem={setCurrentItem}
        data={data}
        searchFilter={searchFilter}
        setSearchFilter={setSearchFilter}
      />
      <ActionSourcesOptions
        currentNamespace={currentNamespace}
        onSelectAllChange={onSelectAllChange}
        selectedApplications={selectedApplications}
        onFutureApplyChange={onFutureApplyChange}
      />
    </SourcesOptionMenuWrapper>
  );
}
