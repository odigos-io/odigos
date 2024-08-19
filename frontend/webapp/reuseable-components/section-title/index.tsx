import React from 'react';
import { Text } from '../text';
import styled from 'styled-components';

interface SectionTitleProps {
  title: string;
  description: string;
  actionButton?: React.ReactNode; // Accept a React node as the action button
}

const Container = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
`;

const TitleContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

const Title = styled(Text)``;

const Description = styled(Text)``;

const SectionTitle: React.FC<SectionTitleProps> = ({
  title,
  description,
  actionButton, // Use the custom action button
}) => {
  return (
    <Container>
      <TitleContainer>
        <Title weight={300} size={20}>
          {title}
        </Title>
        <Description weight={200} opacity={0.8} size={14}>
          {description}
        </Description>
      </TitleContainer>
      {actionButton && <div>{actionButton}</div>}
    </Container>
  );
};

export { SectionTitle };
