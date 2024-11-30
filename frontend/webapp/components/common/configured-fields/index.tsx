import React, { Fragment } from 'react';
import styled from 'styled-components';
import { Text, Status, Tooltip } from '@/reuseable-components';
import { MonitorsLegend } from '@/components/overview';
import theme from '@/styles/theme';

interface Detail {
  title: string;
  tooltip?: string;
  value: string;
}

interface Props {
  details: Detail[];
}

const ListContainer = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 24px 40px;
`;

const ListItem = styled.div``;

const ItemTitle = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  font-size: 10px;
  line-height: 16px;
`;

const ItemValue = styled(Text)`
  color: ${({ theme }) => theme.colors.text};
  font-size: 12px;
  line-height: 18px;
`;

export const ConfiguredFields: React.FC<Props> = ({ details }) => {
  const parseValue = (value: string) => {
    let str = '';

    try {
      const parsed = JSON.parse(value);

      // Handle arrays
      if (Array.isArray(parsed)) {
        str = parsed
          .map((item) => {
            if (typeof item === 'object' && item !== null) return `${item.key}: ${item.value}`;
            else return item;
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

  const renderValue = (title: string, value: string) => {
    switch (title) {
      case 'Status':
        return <Status isActive={value == 'true'} withIcon withBorder withSmaller withSpecialFont />;
      case 'Monitors':
        return <MonitorsLegend color={theme.colors.text} signals={value.split(', ')} />;
      default:
        return <ItemValue>{parseValue(value)}</ItemValue>;
    }
  };

  return (
    <ListContainer>
      {details.map((detail, index) => (
        <ListItem key={index}>
          <Tooltip text={detail.tooltip || ''} withIcon>
            <ItemTitle>{detail.title}</ItemTitle>
          </Tooltip>
          {renderValue(detail.title, detail.value)}
        </ListItem>
      ))}
    </ListContainer>
  );
};
