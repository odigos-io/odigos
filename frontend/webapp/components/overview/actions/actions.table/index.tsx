import { ActionData } from '@/types';
import React, { useState } from 'react';
import { Table } from '@/design.system';
import { ActionsTableRow } from './actions.table.row';
import { ActionsTableHeader } from './actions.table.header';
import { EmptyList } from '@/components/lists';
import { OVERVIEW } from '@/utils';

type TableProps = {
  data: ActionData[];
  onRowClick: (id: string) => void;
  sortActions?: (condition: string) => void;
  filterActionsBySignal?: (signals: string[]) => void;
  toggleActionStatus?: (ids: string[], disabled: boolean) => void;
};

const SELECT_ALL_CHECKBOX = 'select_all';

export const ActionsTable: React.FC<TableProps> = ({
  data,
  onRowClick,
  sortActions,
  filterActionsBySignal,
  toggleActionStatus,
}) => {
  const [selectedCheckbox, setSelectedCheckbox] = useState<string[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage, setItemsPerPage] = useState(10);

  const currentPageRef = React.useRef(1);
  function onSelectedCheckboxChange(id: string) {
    if (id === SELECT_ALL_CHECKBOX) {
      if (selectedCheckbox.length > 0) {
        setSelectedCheckbox([]);
      } else {
        const start = (currentPageRef.current - 1) * 10;
        const end = currentPageRef.current * 10;
        setSelectedCheckbox(data.slice(start, end).map((item) => item.id));
      }
      return;
    }

    if (selectedCheckbox.includes(id)) {
      setSelectedCheckbox(selectedCheckbox.filter((item) => item !== id));
    } else {
      setSelectedCheckbox([...selectedCheckbox, id]);
    }
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

  function renderEmptyResult() {
    return <EmptyList title={OVERVIEW.EMPTY_ACTION} />;
  }

  return (
    <>
      <Table<ActionData>
        data={data}
        renderTableHeader={renderTableHeader}
        onPaginate={onPaginate}
        renderEmptyResult={renderEmptyResult}
        currentPage={currentPage}
        itemsPerPage={itemsPerPage}
        setCurrentPage={setCurrentPage}
        setItemsPerPage={setItemsPerPage}
        renderTableRows={(item, index) => (
          <ActionsTableRow
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
