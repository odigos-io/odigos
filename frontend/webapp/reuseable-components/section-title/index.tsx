import React from 'react';
import Image from 'next/image';
import { Text } from '../text';
import { Badge } from '../badge';
import styled from 'styled-components';

interface SectionTitleProps {
  title: string;
  description: string;
  badgeLabel?: string | number;
  icon?: string;
  actionButton?: React.ReactNode;
  size?: 'small' | 'medium' | 'large';
}

const Container = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
`;

const HeaderWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

const TitleContainer = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const Title = styled(Text)``;

const Description = styled(Text)``;

const SectionTitle: React.FC<SectionTitleProps> = ({ title, description, badgeLabel, icon, actionButton, size = 'medium' }) => {
  const titleSize = size === 'small' ? 16 : size === 'medium' ? 20 : 24;
  const descriptionSize = size === 'small' ? 12 : size === 'medium' ? 14 : 16;

  return (
    <Container>
      <HeaderWrapper>
        <TitleContainer>
          {icon && <Image alt='icon' src={icon} width={14} height={24} />}
          <Title weight={300} size={titleSize}>
            {title}
          </Title>
          {/* we should allow nullish values like "0" to be displayed */}
          {badgeLabel !== undefined && <Badge label={badgeLabel} filled={!!badgeLabel} />}
        </TitleContainer>

        <Description weight={200} opacity={0.8} size={descriptionSize}>
          {description}
        </Description>
      </HeaderWrapper>

      {actionButton && <div>{actionButton}</div>}
    </Container>
  );
};

export { SectionTitle };
