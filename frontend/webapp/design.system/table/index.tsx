import theme from '@/styles/palette';
import { ActionData } from '@/types';
import styled from 'styled-components';
import React, { useState } from 'react';
import { Pagination } from '@/design.system';
import { EmptyList } from '@/components';

type TableProps = {
  data: ActionData[];
  renderTableHeader: () => JSX.Element;
  renderTableRows: (item: any, index: number) => any;
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

export const Table: React.FC<TableProps> = ({
  data,
  renderTableHeader,
  renderTableRows,
  onPaginate,
}) => {
  const [currentPage, setCurrentPage] = useState(1);
  const [itemsPerPage, setItemsPerPage] = useState(10);

  const indexOfLastItem = currentPage * itemsPerPage;
  const indexOfFirstItem = indexOfLastItem - itemsPerPage;
  const currentItems = data.slice(indexOfFirstItem, indexOfLastItem);

  const handlePageChange = (pageNumber) => {
    setCurrentPage(pageNumber);
    onPaginate && onPaginate(pageNumber);
  };

  function renderEmptyResult() {
    return <EmptyList title="No actions found" />;
  }

  return (
    <>
      <StyledTable>
        {renderTableHeader()}
        <StyledTbody>{currentItems.map(renderTableRows)}</StyledTbody>
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
