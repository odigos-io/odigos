import React, { useState } from 'react';
import { Table } from '@/design.system';
import { ManagedSource, Namespace } from '@/types';
import { SourcesTableRow } from './sources.table.row';
import { SourcesTableHeader } from './sources.table.header';
import { EmptyList } from '@/components/lists';
import { OVERVIEW } from '@/utils';

type TableProps = {
  data: ManagedSource[];
  namespaces?: Namespace[];
  onRowClick: (source: ManagedSource) => void;
  sortSources?: (condition: string) => void;
  filterSourcesByKind?: (kinds: string[]) => void;
  filterSourcesByNamespace?: (namespaces: string[]) => void;
  toggleActionStatus?: (ids: string[], disabled: boolean) => void;
  deleteSourcesHandler?: (sources: ManagedSource[]) => void;
};

const SELECT_ALL_CHECKBOX = 'select_all';

export const ManagedSourcesTable: React.FC<TableProps> = ({
  data,
  namespaces,
  onRowClick,
  sortSources,
  toggleActionStatus,
  filterSourcesByKind,
  deleteSourcesHandler,
  filterSourcesByNamespace,
}) => {
  const [selectedCheckbox, setSelectedCheckbox] = useState<string[]>([]);
  const currentPageRef = React.useRef(1);

  function onPaginate(pageNumber: number) {
    currentPageRef.current = pageNumber;
  }

  function renderEmptyResult() {
    return <EmptyList title={OVERVIEW.EMPTY_SOURCE} />;
  }

  function onSelectedCheckboxChange(id: string) {
    const start = (currentPageRef.current - 1) * 10;
    const end = currentPageRef.current * 10;
    const slicedData = data.slice(start, end);
    if (id === SELECT_ALL_CHECKBOX) {
      if (selectedCheckbox.length === slicedData.length) {
        setSelectedCheckbox([]);
      } else {
        setSelectedCheckbox(slicedData.map((item) => JSON.stringify(item)));
      }
      return;
    }

    if (selectedCheckbox.includes(id)) {
      setSelectedCheckbox(selectedCheckbox.filter((item) => item !== id));
    } else {
      setSelectedCheckbox([...selectedCheckbox, id]);
    }
  }

  function parseSelectedSources() {
    const selectedSources = selectedCheckbox.map((item) => JSON.parse(item));
    deleteSourcesHandler && deleteSourcesHandler(selectedSources);
  }

  function renderTableHeader() {
    return (
      <SourcesTableHeader
        data={data}
        namespaces={namespaces}
        sortSources={sortSources}
        toggleActionStatus={toggleActionStatus}
        filterSourcesByKind={filterSourcesByKind}
        filterSourcesByNamespace={filterSourcesByNamespace}
        selectedCheckbox={selectedCheckbox}
        onSelectedCheckboxChange={onSelectedCheckboxChange}
        deleteSourcesHandler={parseSelectedSources}
      />
    );
  }

  return (
    <>
      <Table<ManagedSource>
        data={data}
        renderTableHeader={renderTableHeader}
        onPaginate={onPaginate}
        renderEmptyResult={renderEmptyResult}
        renderTableRows={(item, index) => (
          <SourcesTableRow
            data={data}
            item={item}
            index={index}
            onRowClick={onRowClick}
            selectedCheckbox={selectedCheckbox}
            onSelectedCheckboxChange={onSelectedCheckboxChange}
          />
        )}
      />
    </>
  );
};
