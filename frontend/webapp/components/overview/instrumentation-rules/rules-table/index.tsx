import React, { useState } from 'react';
import { Table } from '@/design.system';
import { InstrumentationRulesTableRow } from './instrumentation.rules.table.row';
import { InstrumentationRulesTableHeader } from './instrumentation.rules.table.header';
import { EmptyList } from '@/components/lists';
import { InstrumentationRuleSpec, RuleData } from '@/types';

type TableProps = {
  data: InstrumentationRuleSpec[];
  onRowClick: (id: string) => void;
  sortRules?: (condition: string) => void;
};

const SELECT_ALL_CHECKBOX = 'select_all';

export const InstrumentationRulesTable: React.FC<TableProps> = ({
  data,
  onRowClick,
  sortRules,
}) => {
  const [selectedCheckbox, setSelectedCheckbox] = useState<string[]>([]);

  const currentPageRef = React.useRef(1);
  function onSelectedCheckboxChange(id: string) {}

  function onPaginate(pageNumber: number) {
    currentPageRef.current = pageNumber;
    selectedCheckbox.length > 0 && setSelectedCheckbox([]);
  }

  function renderTableHeader() {
    return (
      <InstrumentationRulesTableHeader data={data} sortRules={sortRules} />
    );
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
