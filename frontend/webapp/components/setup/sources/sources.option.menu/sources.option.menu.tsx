import React from 'react';
import { SourcesOptionMenuWrapper } from './sources.option.menu.styled';
import { FilterSourcesOptions } from './filter.sources.options';
import { ActionSourcesOptions } from './action.sources.options';
import { KeyvalButton, KeyvalText } from '@/design.system';
import theme from '@/styles/palette';

export function SourcesOptionMenu({
  setCurrentItem,
  data,
  searchFilter,
  setSearchFilter,
  onSelectAllChange,
  selectedApplications,
  currentNamespace,
  onFutureApplyChange,
  toggleFastSourcesSelection,
}: any) {
  return (
    <SourcesOptionMenuWrapper>
      <FilterSourcesOptions
        currentNamespace={currentNamespace}
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
      <KeyvalButton style={{ height: 36 }} onClick={toggleFastSourcesSelection}>
        <KeyvalText color={theme.text.dark_button}>
          Fast Sources Selection
        </KeyvalText>
      </KeyvalButton>
    </SourcesOptionMenuWrapper>
  );
}
