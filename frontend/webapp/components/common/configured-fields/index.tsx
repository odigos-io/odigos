import React, { Fragment } from 'react';
import styled from 'styled-components';
import { Text, Status, Tooltip } from '@/reuseable-components';
import Image from 'next/image';

type TypeDetail = {
  title: string;
  tooltip?: string;
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
  const parseValue = (value: string) => {
    let str = '';

    try {
      const parsed = JSON.parse(value);

      // Handle arrays
      if (Array.isArray(parsed)) {
        str = parsed
          .map((item) => {
            if (typeof item === 'object' && item !== null) {
              return `${item.key}: ${item.value}`;
            }

            return item;
          })
          .join(', ');
      }

      // Handle objects (non-array JSON objects)
      else if (typeof parsed === 'object' && parsed !== null) {
        str = Object.entries(parsed)
          .map(([key, val]) => `${key}: ${val}`)
          .join(', ');
      }

      // Should never reach this if it's a string (it will throw)
      else {
        str = value;
      }
    } catch (error) {
      str = value;
    }

    const strSplitted = str.split('\n');

    return strSplitted.map((str, idx) => (
      <Fragment key={`str-br-${str}-${idx}`}>
        {str}
        {idx < strSplitted.length - 1 ? <br /> : null}
      </Fragment>
    ));
  };

  return (
    <ListContainer>
      {details.map((detail, index) => (
        <ListItem key={index}>
          <Tooltip text={detail.tooltip || ''} withIcon>
            <ItemTitle>{detail.title}</ItemTitle>
          </Tooltip>

          {detail.title === 'Status' ? <Status isActive={detail.value == 'true'} withIcon withBorder withSmaller withSpecialFont /> : <ItemValue>{parseValue(detail.value)}</ItemValue>}
        </ListItem>
      ))}
    </ListContainer>
  );
};
