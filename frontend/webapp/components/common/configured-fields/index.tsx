import React, { Fragment } from 'react';
import styled from 'styled-components';
import { Text } from '@/reuseable-components';

type TypeDetail = {
  title: string;
  value: string;
};

type ConfiguredFieldsProps = {
  details: TypeDetail[];
};

const ListContainer = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 24px 40px;
`;

const ListItem = styled.div``;

const ItemTitle = styled(Text)`
  color: #b8b8b8;
  font-size: 10px;
  line-height: 16px;
`;

const ItemValue = styled(Text)`
  color: ${({ theme }) => theme.colors.text};
  font-size: 12px;
  line-height: 18px;
`;

export const ConfiguredFields: React.FC<ConfiguredFieldsProps> = ({ details }) => {
  const parseValue = (value: any) => {
    try {
      const parsed = JSON.parse(value);
      if (typeof parsed === 'string') {
        return parsed;
      }

      if (Array.isArray(parsed)) {
        return parsed
          .map((item) => {
            if (typeof item === 'object' && item !== null) {
              return `${item.key}: ${item.value}`;
            }

            return item;
          })
          .join(', ');
      }

      // Handle objects (non-array JSON objects)
      if (typeof parsed === 'object' && parsed !== null) {
        return Object.entries(parsed)
          .map(([key, val]) => `${key}: ${val}`)
          .join(', ');
      }
    } catch (error) {
      return value;
    }
    return value;
  };

  return (
    <ListContainer>
      {details.map((detail, index) => (
        <ListItem key={index}>
          <ItemTitle>{detail.title}</ItemTitle>
          <ItemValue>
            {parseValue(detail.value)
              .split('\n')
              .map((str) => (
                <Fragment key={detail.title + str}>
                  {str}
                  <br />
                </Fragment>
              ))}
          </ItemValue>
        </ListItem>
      ))}
    </ListContainer>
  );
};
