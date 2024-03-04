import React from 'react';
import styled from 'styled-components';

import theme from '@/styles/palette';
import { Expand } from '@/assets/icons/app';

type PaginationProps = {
  total: number;
  itemsPerPage: number;
  currentPage: number;
  onPageChange: (page: number) => void;
};

const PaginationContainer = styled.div`
  display: flex;
  width: 100%;
  justify-content: center;
  padding: 20px;
  gap: 2px;
`;

const PageButton = styled.button<{
  isCurrentPage?: boolean;
  isDisabled?: boolean;
}>`
  background-color: ${(props) =>
    props.isCurrentPage ? theme.colors.blue_grey : 'transparent'};
  color: ${(props) => (props.isDisabled ? theme.text.grey : theme.text.white)};
  border: none;
  border-radius: 4px;
  padding: 4px 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;

  &:disabled {
    cursor: default;
  }

  &:hover {
    background-color: ${theme.colors.blue_grey};
  }
`;

// const Expand = styled.svg<{ rotation: number }>`
//   width: 14px;
//   height: 14px;
//   transform: rotate(${(props) => props.rotation}deg);
// `;

export const Pagination: React.FC<PaginationProps> = ({
  total,
  itemsPerPage,
  currentPage,
  onPageChange,
}) => {
  const pageCount = Math.ceil(total / itemsPerPage);

  return (
    <PaginationContainer>
      <PageButton
        onClick={() => onPageChange(currentPage - 1)}
        disabled={currentPage === 1}
        isDisabled={currentPage === 1}
      >
        <Expand style={{ transform: 'rotate(90deg)' }} />
        Previous
      </PageButton>
      {new Array(pageCount).fill(0).map((_, index) => (
        <PageButton
          key={index}
          onClick={() => onPageChange(index + 1)}
          isCurrentPage={currentPage === index + 1}
        >
          {index + 1}
        </PageButton>
      ))}
      <PageButton
        onClick={() => onPageChange(currentPage + 1)}
        disabled={currentPage === pageCount}
        isDisabled={currentPage === pageCount}
      >
        Next
        <Expand style={{ transform: 'rotate(-90deg)' }} />
      </PageButton>
    </PaginationContainer>
  );
};
