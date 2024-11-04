'use client';
import React, { useEffect, useState } from 'react';
import { Describe, Refresh } from '@/assets';
import theme from '@/styles/palette';
import { useDescribe } from '@/hooks';
import styled from 'styled-components';
import { Drawer, KeyvalText } from '@/design.system';

interface SourceDescriptionDrawerProps {
  namespace: string;
  kind: string;
  name: string;
}

interface DescribeItem {
  name: string;
  value: string;
  explain?: string;
}

export const SourceDescriptionDrawer: React.FC<
  SourceDescriptionDrawerProps
> = ({ namespace, kind, name }) => {
  const [isOpen, setDrawerOpen] = useState(false);
  const [badgeStatus, setBadgeStatus] = useState<
    'error' | 'transitioning' | 'success'
  >('success');

  const toggleDrawer = () => setDrawerOpen(!isOpen);

  const {
    sourceDescription,
    isSourceLoading,
    fetchSourceDescription,
    setNamespaceKindName,
  } = useDescribe();

  useEffect(() => {
    isOpen &&
      namespace &&
      kind &&
      name &&
      setNamespaceKindName(namespace, kind, name);
  }, [isOpen, namespace, kind, name]);

  useEffect(() => {
    if (sourceDescription) {
      const statuses = extractSourceStatuses(sourceDescription);
      if (statuses.includes('error')) setBadgeStatus('error');
      else if (statuses.includes('transitioning'))
        setBadgeStatus('transitioning');
      else setBadgeStatus('success');
    }
  }, [sourceDescription]);

  return (
    <>
      <IconWrapper onClick={toggleDrawer}>
        <Describe style={{ cursor: 'pointer' }} size={10} />
        {!isSourceLoading && (
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

      {isOpen && (
        <Drawer
          isOpen={isOpen}
          onClose={() => setDrawerOpen(false)}
          position="right"
          width="fit-content"
        >
          {isSourceLoading ? (
            <LoadingMessage>Loading source details...</LoadingMessage>
          ) : (
            <DescriptionContent>
              {sourceDescription
                ? formatDescription(sourceDescription, () =>
                    fetchSourceDescription()
                  )
                : 'No source details available.'}
            </DescriptionContent>
          )}
        </Drawer>
      )}
    </>
  );
};

function extractSourceStatuses(description: any): string[] {
  const statuses: string[] = [];
  if (description.instrumentationConfig?.status) {
    statuses.push(description.instrumentationConfig.status);
  }
  description.pods?.forEach((pod: any) => {
    if (pod.phase.status) statuses.push(pod.phase.status);
  });
  return statuses;
}

// Generic function to format any description data
function formatDescription(description: any, refetch: () => void) {
  const renderObjectProperties = (obj: any) => {
    return Object.entries(obj).map(([key, item]: [string, DescribeItem]) => {
      if (
        typeof item === 'object' &&
        item !== null &&
        item.hasOwnProperty('value') &&
        item.hasOwnProperty('name')
      ) {
        return (
          <div key={key}>
            <p>
              <strong>- {item?.name}:</strong> {String(item.value)}
            </p>
            {item.explain && <ExplanationText>{item.explain}</ExplanationText>}
          </div>
        );
      } else if (typeof item === 'object' && item !== null) {
        return (
          <div key={key} style={{ marginLeft: '16px' }}>
            {renderObjectProperties(item)}
          </div>
        );
      } else if (Array.isArray(item)) {
        return <CollectorSection key={key} title={key} collector={item} />;
      }
      return null;
    });
  };

  return (
    <div>
      <VersionHeader>
        <VersionText>{description.name?.value || 'Unnamed'}</VersionText>
        <IconWrapper onClick={refetch}>
          <Refresh size={16} />
        </IconWrapper>
      </VersionHeader>
      {renderObjectProperties(description)}
    </div>
  );
}
// Component to handle pod data display
const CollectorSection: React.FC<{ title: string; collector: any[] }> = ({
  title,
  collector,
}) => (
  <section style={{ marginTop: 24 }}>
    <CollectorTitle>{title}</CollectorTitle>
    {collector.map((item: any, index: number) => (
      <CollectorItem
        key={index}
        label={item.podName.value}
        value={item.phase.value}
        status={item.phase.status}
      />
    ))}
  </section>
);

// Component to handle individual pod items with conditional styling based on status
const CollectorItem: React.FC<{
  label: string;
  value: any;
  status?: string;
}> = ({ label, value, status }) => {
  const color = status === 'error' ? theme.colors.error : theme.text.light_grey;

  return (
    <StatusText color={color}>
      - {label}: {String(value)}
    </StatusText>
  );
};

const VersionHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
`;

const VersionText = styled(KeyvalText)`
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
  cursor: pointer;
  &:hover {
    background-color: ${theme.colors.dark};
  }
`;

const LoadingMessage = styled.p`
  font-size: 1rem;
  color: #555;
`;

const DescriptionContent = styled(KeyvalText)`
  white-space: pre-wrap;
  line-height: 1.6;
  padding: 20px;
`;

const StatusText = styled.div<{ color: string }>`
  color: ${({ color }) => color};
  font-weight: bold;
  margin-bottom: 8px;
  padding-left: 16px;
`;

const ExplanationText = styled.p`
  font-size: 0.9rem;
  color: ${theme.text.light_grey};
  margin-top: -5px;
  margin-bottom: 10px;
`;
