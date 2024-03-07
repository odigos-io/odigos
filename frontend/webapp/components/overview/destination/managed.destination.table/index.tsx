import React from 'react';
import { Table } from '@/design.system';
import { Destination } from '@/types';
import { EmptyList } from '@/components/lists';
import { OVERVIEW } from '@/utils';
import { DestinationsTableHeader } from './destinations.table.header';
import { DestinationsTableRow } from './destinations.table.row';

type TableProps = {
  data: Destination[];
  onRowClick: (source: Destination) => void;
};

export const ManagedDestinationsTable: React.FC<TableProps> = ({
  data,
  onRowClick,
}) => {
  const currentPageRef = React.useRef(1);

  function onPaginate(pageNumber: number) {
    currentPageRef.current = pageNumber;
  }

  function renderEmptyResult() {
    return <EmptyList title={OVERVIEW.EMPTY_SOURCE} />;
  }

  function renderTableHeader() {
    return <DestinationsTableHeader data={data} />;
  }

  return (
    <>
      <Table<Destination>
        data={data}
        renderTableHeader={renderTableHeader}
        onPaginate={onPaginate}
        renderEmptyResult={renderEmptyResult}
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
