import React from 'react';
import { SVG } from '@/assets';
import { isEmpty } from '@/utils';
import styled from 'styled-components';
import { IconWrapped } from '../icon-wrapped';

interface ColumnCell {
  key: string; // used to bind the row cell to the column
  title: string;
}

interface RowCell {
  columnKey: string; // used to bind the row cell to the column
  icon?: SVG;
  value?: string;
  component?: () => React.ReactNode;
}

interface Props {
  columns: ColumnCell[];
  rows: RowCell[][];
}

const Container = styled.div`
  position: relative;
  width: 100%;
`;

const Table = styled.table`
  border-collapse: collapse;
  width: 100%;
`;

const TableHead = styled.thead`
  border-top: 1px solid ${({ theme }) => theme.colors.white_opacity['008']};
  border-bottom: 1px solid ${({ theme }) => theme.colors.white_opacity['008']};
`;

const TableTitle = styled.th`
  padding: 8px;
  color: ${({ theme }) => theme.text.darker_grey};
  font-family: ${({ theme }) => theme.font_family.secondary};
  text-transform: uppercase;
  font-size: 12px;
  font-weight: 500;
  text-align: left;
`;

const TableBody = styled.tbody``;

const TableRow = styled.tr`
  line-height: 68px;
`;

const TableData = styled.td<{ $isFirst: boolean }>`
  padding: 12px 8px 0 ${({ $isFirst }) => ($isFirst ? '16px' : '8px')};
  color: ${({ theme }) => theme.text.secondary};
  font-family: ${({ theme }) => theme.font_family.primary};
  font-size: 14px;
  white-space: nowrap;
  width: fit-content;
`;

const RowBackground = styled.div<{ $index: number }>`
  position: absolute;
  top: ${({ $index }) => $index * 80 + 34}px;
  left: 0;
  z-index: -1;
  width: 100%;
  height: 68px;
  margin: 12px 0;
  border-radius: 16px;
  background-color: ${({ theme }) => theme.colors.white_opacity['004']};
`;

export const InteractiveTable: React.FC<Props> = ({ columns, rows }) => {
  return (
    <Container>
      <Table>
        <TableHead>
          <tr>
            {columns.map(({ key, title }) => (
              <TableTitle key={`column-${key}`}>{title}</TableTitle>
            ))}
          </tr>
        </TableHead>

        <TableBody>
          {rows.map((row, i) => (
            <TableRow key={`row-${i}`}>
              {columns.map(({ key }, ii) => {
                const rowCell = row.find(({ columnKey }) => columnKey === key);
                if (!rowCell) return null;
                const { value, icon: Icon, component: Component } = rowCell;

                return (
                  <TableData key={`row-${i}-cell-${key}`} $isFirst={ii === 0}>
                    {!isEmpty(value) ? value : !!Icon ? <IconWrapped icon={Icon} /> : !!Component ? <Component /> : '-'}
                  </TableData>
                );
              })}
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {rows.map((_, i) => (
        <RowBackground key={`bg-${i}`} $index={i} />
      ))}
    </Container>
  );
};
