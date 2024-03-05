import { ActionData } from '@/types';
import React, { useState } from 'react';
import { Table } from '@/design.system';
import { ActionsTableHeader } from './actions.table.header';
import { ActionsTableRow } from './actions.table.row';

type TableProps = {
  data: ActionData[];
  onRowClick: (id: string) => void;
  sortActions?: (condition: string) => void;
  filterActionsBySignal?: (signals: string[]) => void;
};

const SELECT_ALL_CHECKBOX = 'select_all';

export const ActionsTable: React.FC<TableProps> = ({
  data,
  onRowClick,
  sortActions,
  filterActionsBySignal,
}) => {
  const [selectedCheckbox, setSelectedCheckbox] = useState<string[]>([]);

  function onSelectedCheckboxChange(id: string) {
    if (id === SELECT_ALL_CHECKBOX) {
      if (selectedCheckbox.length === data.length) {
        setSelectedCheckbox([]);
      } else {
        setSelectedCheckbox(data.map((item) => item.id));
      }
      return;
    }

    if (selectedCheckbox.includes(id)) {
      setSelectedCheckbox(selectedCheckbox.filter((item) => item !== id));
    } else {
      setSelectedCheckbox([...selectedCheckbox, id]);
    }
  }

  function renderTableHeader() {
    return (
      <ActionsTableHeader
        data={data}
        selectedCheckbox={selectedCheckbox}
        onSelectedCheckboxChange={onSelectedCheckboxChange}
        sortActions={sortActions}
        filterActionsBySignal={filterActionsBySignal}
      />
    );
  }

  return (
    <>
      <Table
        data={data}
        renderTableHeader={renderTableHeader}
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
