import React from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { MONITORS_OPTIONS } from '@/utils';
import { Text } from '@/reuseable-components';

const List = styled.div`
  display: flex;
  align-items: center;
  gap: 24px;
`;

const ListItem = styled.div`
  display: flex;
  align-items: center;
  gap: 4px;
`;

const MonitorsLegend = () => {
  return (
    <List>
      {MONITORS_OPTIONS.map(({ id, value }) => (
        <ListItem key={`monitors-legend-${id}`}>
          <Image
            src={`/icons/monitors/${id}.svg`}
            width={14}
            height={14}
            alt={value}
            style={{ filter: 'invert(40%) brightness(80%) grayscale(100%)' }}
          />

          <Text size={14} color={theme.text.grey}>
            {value}
          </Text>
        </ListItem>
      ))}
    </List>
  );
};

export { MonitorsLegend };
