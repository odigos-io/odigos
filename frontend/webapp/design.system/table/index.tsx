import React from 'react';
import { Table } from '@odigos-io/design-system';

// Updated TableProps to be generic
type TableProps<T> = {
  data: T[];
  renderTableHeader: () => JSX.Element;
  renderTableRows: (item: T, index: number) => JSX.Element;
  renderEmptyResult: () => JSX.Element;
  onPaginate?: (pageNumber: number) => void;
};

export const OdigosTable = <T,>(props: TableProps<T>) => {
  return <Table<T> {...props} />;
};
