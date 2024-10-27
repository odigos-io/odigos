'use client';
import React, { useEffect, useState } from 'react';
import { Describe } from '@/assets';
import theme from '@/styles/palette';
import { useDescribe } from '@/hooks';
import styled from 'styled-components';
import { Drawer, KeyvalText } from '@/design.system';

interface OdigosDescriptionDrawerProps {}

export const OdigosDescriptionDrawer: React.FC<
  OdigosDescriptionDrawerProps
> = ({}) => {
  const [isOpen, setDrawerOpen] = useState(false);

  const toggleDrawer = () => setDrawerOpen(!isOpen);

  const { odigosDescription, isOdigosLoading, refetchOdigosDescription } =
    useDescribe();

  useEffect(() => {
    if (isOpen) {
      refetchOdigosDescription();
    }
  }, [isOpen, refetchOdigosDescription]);

  return (
    <>
      <IconWrapper>
        <Describe
          style={{ cursor: 'pointer' }}
          size={10}
          onClick={toggleDrawer}
        />
      </IconWrapper>

      <Drawer
        isOpen={isOpen}
        onClose={() => setDrawerOpen(false)}
        position="right"
        width="auto"
      >
        {isOdigosLoading ? (
          <LoadingMessage>Loading description...</LoadingMessage>
        ) : (
          <DescriptionContent>
            {odigosDescription
              ? formatOdigosDescription(odigosDescription)
              : 'No description available.'}
          </DescriptionContent>
        )}
      </Drawer>
    </>
  );
};

const IconWrapper = styled.div`
  position: relative;
  padding: 8px;
  width: 16px;
  border-radius: 8px;
  border: 1px solid ${theme.colors.blue_grey};
  display: flex;
  align-items: center;
  &:hover {
    background-color: ${theme.colors.dark};
  }
`;

// Styled component for loading message
const LoadingMessage = styled.p`
  font-size: 1rem;
  color: #555;
`;

// Styled component for description content
const DescriptionContent = styled(KeyvalText)`
  white-space: pre-wrap;
  line-height: 1.6;
  padding: 20px;
`;

function formatOdigosDescription(description: string) {
  const lines = description.split('\n');
  return (
    <div>
      {lines.map((line, index) => (
        <div key={index}>{applyStatusColor(line)}</div>
      ))}
    </div>
  );
}

function applyStatusColor(line: string) {
  if (line.includes('Odigos Version')) return <h3>{line}</h3>;

  if (line.includes('Collectors Group Created')) {
    return (
      <StatusText color="green">{line.replace(/\[32m|\[0m/g, '')}</StatusText>
    );
  }
  if (
    line.includes('Deployed: Status Unavailable') ||
    line.includes('Failed')
  ) {
    return (
      <StatusText color="red">{line.replace(/\[31m|\[0m/g, '')}</StatusText>
    );
  }
  if (
    line.includes('Ready: true') ||
    line.includes('Deployment: Found') ||
    line.includes('Current Number')
  ) {
    return (
      <StatusText color="green">{line.replace(/\[32m|\[0m/g, '')}</StatusText>
    );
  }

  return <span>{line}</span>;
}

const StatusText = styled.span<{ color: string }>`
  color: ${({ color }) => color};
  font-weight: bold;
`;
