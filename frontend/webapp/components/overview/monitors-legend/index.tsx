import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { MONITORS_OPTIONS } from '@/utils';
import { Text } from '@/reuseable-components';

interface Props {
  size?: number;
  color?: string;
  signals?: string[];
}

const List = styled.div<{ $size: number }>`
  display: flex;
  align-items: center;
  gap: ${({ $size }) => $size}px;
`;

const ListItem = styled.div<{ $size: number }>`
  display: flex;
  align-items: center;
  gap: ${({ $size }) => $size / 3}px;
`;

const MonitorTitle = styled(Text)<{ $size: number; $color?: string }>`
  color: ${({ $color, theme }) => $color || theme.text.grey};
  font-size: ${({ $size }) => $size}px;
  font-weight: 300;
  line-height: 150%;
`;

export const MonitorsLegend: React.FC<Props> = ({ size = 12, color, signals }) => {
  return (
    <List $size={size}>
      {MONITORS_OPTIONS.map(({ id, value }) => {
        const ok = !signals || !signals.length || signals.find((str) => str.toLowerCase() === id);

        if (!ok) return null;

        return (
          <ListItem key={`monitors-legend-${id}`} $size={size}>
            <Image src={`/icons/monitors/${id}.svg`} width={size + 2} height={size + 2} alt={value} style={{ filter: 'invert(40%) brightness(80%) grayscale(100%)' }} />
            <MonitorTitle $size={size} $color={color}>
              {value}
            </MonitorTitle>
          </ListItem>
        );
      })}
    </List>
  );
};
