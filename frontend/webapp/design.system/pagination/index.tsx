import React from 'react';
import { Pagination } from '@odigos-io/design-system';

type PaginationProps = {
  total: number;
  itemsPerPage: number;
  currentPage: number;
  onPageChange: (page: number) => void;
};

export const OdigosPagination: React.FC<PaginationProps> = (props) => {
  return <Pagination {...props} />;
};
