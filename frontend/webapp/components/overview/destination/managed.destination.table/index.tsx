import React, { useState } from 'react';
import { Table } from '@/design.system';
import { Destination } from '@/types';
import { EmptyList } from '@/components/lists';
import { OVERVIEW } from '@/utils';
import { DestinationsTableHeader } from './destinations.table.header';
import { DestinationsTableRow } from './destinations.table.row';

type TableProps = {
  data: Destination[];
  onRowClick: (source: Destination) => void;
  sortDestinations?: (condition: string) => void;
  filterDestinationsBySignal?: (signals: string[]) => void;
};

export const ManagedDestinationsTable: React.FC<TableProps> = ({
  data,
  onRowClick,
  sortDestinations,
  filterDestinationsBySignal,
}) => {
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage, setItemsPerPage] = useState(10);
  const currentPageRef = React.useRef(1);

  function onPaginate(pageNumber: number) {
    currentPageRef.current = pageNumber;
  }

  function renderEmptyResult() {
    return <EmptyList title={OVERVIEW.EMPTY_SOURCE} />;
  }

  function renderTableHeader() {
    return (
      <DestinationsTableHeader
        data={data}
        sortDestinations={sortDestinations}
        filterDestinationsBySignal={filterDestinationsBySignal}
      />
    );
  }

  return (
    <>
      <Table<Destination>
        data={data}
        renderTableHeader={renderTableHeader}
        onPaginate={onPaginate}
        renderEmptyResult={renderEmptyResult}
        currentPage={currentPage}
        itemsPerPage={itemsPerPage}
        setCurrentPage={setCurrentPage}
        setItemsPerPage={setItemsPerPage}
        renderTableRows={(item, index) => (
          <DestinationsTableRow
            onRowClick={onRowClick}
            item={item}
            index={index}
          />
        )}
      />
    </>
  );
};
