import { Text } from '@/reuseable-components';
import { DestinationTypeDetail } from '@/types';
import React from 'react';
import styled from 'styled-components';

type ConfiguredDestinationFieldsProps = {
  details: DestinationTypeDetail[];
};

const ListContainer = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 40px;
`;

const ListItem = styled.div``;

const ItemTitle = styled(Text)`
  color: #b8b8b8;
  font-size: 10px;
  line-height: 16px;
`;

const ItemValue = styled(Text)`
  word-break: break-all;
  width: 90%;
  color: ${({ theme }) => theme.colors.text};
  font-size: 12px;
  line-height: 18px;
`;

export const ConfiguredDestinationFields: React.FC<
  ConfiguredDestinationFieldsProps
> = ({ details }) => {
  return (
    <ListContainer>
      {details.map((detail, index) => (
        <ListItem key={index}>
          <ItemTitle>{detail.title}</ItemTitle>
          <ItemValue>{detail.value}</ItemValue>
        </ListItem>
      ))}
    </ListContainer>
  );
};
