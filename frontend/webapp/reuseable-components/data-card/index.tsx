import React from 'react';
import styled from 'styled-components';
import { Badge, Text } from '@/reuseable-components';
import { DataCardFields, type DataCardRow } from './data-card-fields';
export { DataCardFields, type DataCardRow };

interface Props {
  title?: string;
  titleBadge?: string | number;
  description?: string;
  data: DataCardRow[];
}

const CardContainer = styled.div`
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  align-self: stretch;
  gap: 16px;
  padding: 24px;
  border-radius: 24px;
  border: 1px solid ${({ theme }) => theme.colors.border};
`;

const Header = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

const Title = styled(Text)`
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
`;

const Description = styled(Text)`
  font-size: 12px;
  color: ${({ theme }) => theme.text.grey};
`;

export const DataCard: React.FC<Props> = ({ title = 'Details', titleBadge, description, data }) => {
  return (
    <CardContainer>
      <Header>
        <Title>
          {title}
          {/* NOT undefined, because we should allow zero (0) values */}
          {titleBadge !== undefined && <Badge label={titleBadge} />}
        </Title>
        {!!description && <Description>{description}</Description>}
      </Header>

      <DataCardFields data={data} />
    </CardContainer>
  );
};
