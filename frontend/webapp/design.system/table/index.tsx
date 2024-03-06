import theme from '@/styles/palette';
import styled from 'styled-components';
import React, { useState } from 'react';
import { Pagination } from '@/design.system';
import { EmptyList } from '@/components';

// Updated TableProps to be generic
type TableProps<T> = {
  data: T[];
  renderTableHeader: () => JSX.Element;
  renderTableRows: (item: T, index: number) => JSX.Element;
  onPaginate?: (pageNumber: number) => void;
};

const StyledTable = styled.table`
  width: 100%;
  background-color: ${theme.colors.dark};
  border: 1px solid ${theme.colors.blue_grey};
  border-radius: 6px;
  overflow: hidden;
  border-collapse: separate;
  border-spacing: 0;
`;

const StyledTbody = styled.tbody``;

// Applying generic type T to the Table component
export const Table = <T,>({
  data,
  renderTableHeader,
  renderTableRows,
  onPaginate,
}: TableProps<T>) => {
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage, setItemsPerPage] = useState(10);

  const indexOfLastItem = currentPage * itemsPerPage;
  const indexOfFirstItem = indexOfLastItem - itemsPerPage;
  const currentItems = data.slice(indexOfFirstItem, indexOfLastItem);

  const handlePageChange = (pageNumber: number) => {
    setCurrentPage(pageNumber);
    if (onPaginate) {
      onPaginate(pageNumber);
    }
  };

  function renderEmptyResult() {
    return <EmptyList title="No actions found" />;
  }

  return (
    <>
      <StyledTable>
        {renderTableHeader()}
        <StyledTbody>
          {currentItems.map((item, index) => renderTableRows(item, index))}
        </StyledTbody>
      </StyledTable>

      {data.length === 0 ? (
        renderEmptyResult()
      ) : (
        <Pagination
          total={data.length}
          itemsPerPage={itemsPerPage}
          currentPage={currentPage}
          onPageChange={handlePageChange}
        />
      )}
    </>
  );
};
