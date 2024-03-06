import React, { useState } from 'react';
import { Table } from '@/design.system';
import { ManagedSource } from '@/types';
import { SourcesTableRow } from './sources.table.row';
import { ActionsTableHeader } from './sources.table.header';

type TableProps = {
  data: ManagedSource[];
  onRowClick: (source: ManagedSource) => void;
  sortActions?: (condition: string) => void;
  filterActionsBySignal?: (signals: string[]) => void;
  toggleActionStatus?: (ids: string[], disabled: boolean) => void;
};

const SELECT_ALL_CHECKBOX = 'select_all';

export const ManagedSourcesTable: React.FC<TableProps> = ({
  data,
  onRowClick,
  sortActions,
  filterActionsBySignal,
  toggleActionStatus,
}) => {
  const [selectedCheckbox, setSelectedCheckbox] = useState<string[]>([]);

  const currentPageRef = React.useRef(1);
  function onSelectedCheckboxChange(id: string) {
    // if (id === SELECT_ALL_CHECKBOX) {
    //   if (selectedCheckbox.length > 0) {
    //     setSelectedCheckbox([]);
    //   } else {
    //     const start = (currentPageRef.current - 1) * 10;
    //     const end = currentPageRef.current * 10;
    //     setSelectedCheckbox(data.slice(start, end).map((item) => item.id));
    //   }
    //   return;
    // }
    // if (selectedCheckbox.includes(id)) {
    //   setSelectedCheckbox(selectedCheckbox.filter((item) => item !== id));
    // } else {
    //   setSelectedCheckbox([...selectedCheckbox, id]);
    // }
  }

  function onPaginate(pageNumber: number) {
    currentPageRef.current = pageNumber;
    selectedCheckbox.length > 0 && setSelectedCheckbox([]);
  }

  function renderTableHeader() {
    return (
      <ActionsTableHeader
        data={data}
        selectedCheckbox={selectedCheckbox}
        onSelectedCheckboxChange={onSelectedCheckboxChange}
        sortActions={sortActions}
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
            selectedCheckbox={selectedCheckbox}
            onSelectedCheckboxChange={onSelectedCheckboxChange}
            data={data}
            item={item}
            index={index}
          />
        )}
      />
    </>
  );
};
