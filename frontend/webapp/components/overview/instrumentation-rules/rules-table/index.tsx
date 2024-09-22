import React, { useState } from 'react';
import { Table } from '@/design.system';
import { EmptyList } from '@/components/lists';
import { InstrumentationRuleSpec } from '@/types';
import { InstrumentationRulesTableRow } from './instrumentation.rules.table.row';
import { InstrumentationRulesTableHeader } from './instrumentation.rules.table.header';

type TableProps = {
  data: InstrumentationRuleSpec[];
  onRowClick: (id: string) => void;
};

export const InstrumentationRulesTable: React.FC<TableProps> = ({
  data,
  onRowClick,
}) => {
  const [selectedCheckbox, setSelectedCheckbox] = useState<string[]>([]);

  const currentPageRef = React.useRef(1);

  function onPaginate(pageNumber: number) {
    currentPageRef.current = pageNumber;
    selectedCheckbox.length > 0 && setSelectedCheckbox([]);
  }

  function renderTableHeader() {
    return <InstrumentationRulesTableHeader data={data} />;
  }

  function renderEmptyResult() {
    return <EmptyList title={'No rules found'} />;
  }

  return (
    <>
      <Table<InstrumentationRuleSpec>
        data={data}
        renderTableHeader={renderTableHeader}
        onPaginate={onPaginate}
        renderEmptyResult={renderEmptyResult}
        renderTableRows={(item, index) => (
          <InstrumentationRulesTableRow
            item={item}
            index={index}
            onRowClick={onRowClick}
          />
        )}
      />
    </>
  );
};
