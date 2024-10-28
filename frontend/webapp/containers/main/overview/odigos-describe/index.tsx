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
  const [badgeStatus, setBadgeStatus] = useState<
    'error' | 'transitioning' | 'success'
  >('success');

  const toggleDrawer = () => setDrawerOpen(!isOpen);

  const { odigosDescription, isOdigosLoading, refetchOdigosDescription } =
    useDescribe();

  useEffect(() => {
    if (odigosDescription) {
      // Evaluate statuses to set badge based on current data
      const statuses = extractStatuses(odigosDescription);
      if (statuses.includes('error')) setBadgeStatus('error');
      else if (statuses.includes('transitioning'))
        setBadgeStatus('transitioning');
      else setBadgeStatus('success');
    }
  }, [odigosDescription]);

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
        {!isOdigosLoading && (
          <NotificationBadge status={badgeStatus}>
            <KeyvalText size={10}>
              {badgeStatus === 'transitioning'
                ? '...'
                : badgeStatus === 'error'
                ? '!'
                : ''}
            </KeyvalText>
          </NotificationBadge>
        )}
      </IconWrapper>

      <Drawer
        isOpen={isOpen}
        onClose={() => setDrawerOpen(false)}
        position="right"
        width="500px"
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

// Function to extract statuses from the odigosDescription response
function extractStatuses(description: any): string[] {
  const statuses: string[] = [];
  Object.values(description.clusterCollector).forEach((item: any) => {
    if (item.status) statuses.push(item.status);
  });
  Object.values(description.nodeCollector).forEach((item: any) => {
    if (item.status) statuses.push(item.status);
  });
  return statuses;
}

// Render the description with status-specific styling
function formatOdigosDescription(description: any) {
  return (
    <div>
      {/* Display Odigos Version */}
      {description.odigosVersion && (
        <VersionText>
          {description.odigosVersion.name}: {description.odigosVersion.value}
        </VersionText>
      )}

      {/* Display Destinations and Sources Count */}
      <p>Destinations: {description.numberOfDestinations}</p>
      <p>Sources: {description.numberOfSources}</p>

      {/* Display Cluster Collector */}
      <CollectorSection
        title="Cluster Collector"
        collector={description.clusterCollector}
      />

      {/* Display Node Collector */}
      <CollectorSection
        title="Node Collector"
        collector={description.nodeCollector}
      />
    </div>
  );
}

// Component to handle collector data (cluster and node collectors)
const CollectorSection: React.FC<{ title: string; collector: any }> = ({
  title,
  collector,
}) => (
  <section style={{ marginTop: 24 }}>
    <CollectorTitle>{title}</CollectorTitle>
    {Object.entries(collector).map(([key, value]: [string, any]) => (
      <CollectorItem
        key={key}
        label={value.name}
        value={value.value}
        status={value.status}
      />
    ))}
  </section>
);

// Component to handle individual collector items with conditional styling based on status
const CollectorItem: React.FC<{
  label: string;
  value: any;
  status?: string;
}> = ({ label, value, status }) => {
  const color =
    status === 'error'
      ? 'red'
      : status === 'transitioning'
      ? 'orange'
      : status === 'success'
      ? 'green'
      : 'inherit';

  return (
    <StatusText color={color}>
      {label}: {String(value)} {status && <StatusBadge>{status}</StatusBadge>}
    </StatusText>
  );
};

const VersionText = styled(KeyvalText)`
  margin-bottom: 10px;
  font-size: 24px;
`;

const CollectorTitle = styled(KeyvalText)`
  font-size: 20px;
  margin-bottom: 10px;
`;

const NotificationBadge = styled.div<{ status: string }>`
  position: absolute;
  top: -4px;
  right: -4px;
  background-color: ${({ status }) =>
    status === 'error'
      ? theme.colors.error
      : status === 'transitioning'
      ? theme.colors.orange_brown
      : theme.colors.success};
  color: white;
  border-radius: 50%;
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
`;

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

const StatusText = styled.div<{ color: string }>`
  color: ${({ color }) => color};
  font-weight: bold;
  margin-bottom: 8px;
`;

const StatusBadge = styled.span`
  font-size: 0.8rem;
  font-weight: normal;
  margin-left: 4px;
  color: inherit;
`;
