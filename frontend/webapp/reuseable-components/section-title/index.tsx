import React from 'react';
import { Text } from '../text';
import { Button } from '../button';
import styled from 'styled-components';

interface SectionTitleProps {
  title: string;
  description: string;
  buttonText?: string;
  onButtonClick?: () => void;
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

const ActionButton = styled(Button)``;

const ActionButtonText = styled(Text)`
  font-family: ${({ theme }) => theme.font_family.secondary};
  font-weight: 500;
  text-decoration: underline;
  text-transform: uppercase;
  font-size: 14px;
  line-height: 157.143%;
`;

const SectionTitle: React.FC<SectionTitleProps> = ({
  title,
  description,
  buttonText,
  onButtonClick,
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
      {buttonText && onButtonClick && (
        <ActionButton variant={'secondary'} onClick={onButtonClick}>
          <ActionButtonText size={14}>{buttonText}</ActionButtonText>
        </ActionButton>
      )}
    </Container>
  );
};

export { SectionTitle };
