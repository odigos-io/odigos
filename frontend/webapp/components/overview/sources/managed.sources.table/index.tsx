import React, { useState } from 'react';
import { Table } from '@/design.system';
import { ManagedSource } from '@/types';
import { SourcesTableRow } from './sources.table.row';
import { SourcesTableHeader } from './sources.table.header';

type TableProps = {
  data: ManagedSource[];
  onRowClick: (source: ManagedSource) => void;
  sortSources?: (condition: string) => void;
  filterActionsBySignal?: (signals: string[]) => void;
  toggleActionStatus?: (ids: string[], disabled: boolean) => void;
};

export const ManagedSourcesTable: React.FC<TableProps> = ({
  data,
  onRowClick,
  sortSources,
  filterActionsBySignal,
  toggleActionStatus,
}) => {
  const currentPageRef = React.useRef(1);

  function onPaginate(pageNumber: number) {
    currentPageRef.current = pageNumber;
  }

  function renderTableHeader() {
    return (
      <SourcesTableHeader
        data={data}
        sortSources={sortSources}
        filterActionsBySignal={filterActionsBySignal}
        toggleActionStatus={toggleActionStatus}
      />
    );
  }

  return (
    <>
      <Table<ManagedSource>
        data={data}
        renderTableHeader={renderTableHeader}
        onPaginate={onPaginate}
        renderTableRows={(item, index) => (
          <SourcesTableRow
            onRowClick={onRowClick}
            data={data}
            item={item}
            index={index}
          />
        )}
      />
    </>
  );
};
