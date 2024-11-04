import React, { useEffect, useState } from 'react';
import theme from '@/styles/palette';
import { useDescribe } from '@/hooks';
import styled from 'styled-components';
import { Drawer, KeyvalText } from '@/design.system';

interface SourceDescriptionDrawerProps {
  namespace: string;
  kind: string;
  name: string;
}

export const SourceDescriptionDrawer: React.FC<
  SourceDescriptionDrawerProps
> = ({ namespace, kind, name }) => {
  const [isOpen, setDrawerOpen] = useState(false);

  const toggleDrawer = () => setDrawerOpen((prev) => !prev);

  const { sourceDescription, isSourceLoading, fetchSourceDescription } =
    useDescribe();

  useEffect(() => {
    if (!sourceDescription && isOpen) {
      fetchSourceDescription(namespace, kind, name);
    }
  }, [sourceDescription, isOpen]);

  return (
    <>
      <IconWrapper onClick={toggleDrawer}>
        <KeyvalText size={14} color={theme.colors.dark_blue}>
          View Source
        </KeyvalText>
      </IconWrapper>

      <Drawer
        isOpen={isOpen}
        onClose={() => setDrawerOpen(false)}
        position="right"
        width="fit-content"
      >
        {isSourceLoading ? (
          <LoadingMessage>Loading source description...</LoadingMessage>
        ) : (
          <DescriptionContent>
            {sourceDescription
              ? formatSourceDescription(sourceDescription)
              : 'No description available.'}
          </DescriptionContent>
        )}
      </Drawer>
    </>
  );
};

// Function to render the source description with relevant details
function formatSourceDescription(description: any) {
  return (
    <div>
      <Section>
        <KeyvalText>
          <strong>{description.name.name}:</strong> {description.name.value}
        </KeyvalText>
        <ToggleExplanation text={description.name.explain} />
      </Section>
      <Section>
        <KeyvalText>
          <strong>{description.kind.name}:</strong> {description.kind.value}
        </KeyvalText>
        <ToggleExplanation text={description.kind.explain} />
      </Section>
      <Section>
        <KeyvalText>
          <strong>{description.namespace.name}:</strong>{' '}
          {description.namespace.value}
        </KeyvalText>
        <ToggleExplanation text={description.namespace.explain} />
      </Section>

      <LabelsSection title="Labels" labels={description.labels} />
      <InstrumentationSection
        title="Instrumentation Config"
        config={description.instrumentationConfig}
      />
      <RuntimeInfoSection runtimeInfo={description.runtimeInfo} />
      <PodsSection
        pods={description.pods}
        podsPhasesCount={description.podsPhasesCount}
        totalPods={description.totalPods}
      />
    </div>
  );
}

// ToggleExplanation component to handle the show/hide logic
const ToggleExplanation: React.FC<{ text: string }> = ({ text }) => {
  const [isVisible, setVisible] = useState(false);

  return (
    <div>
      <SeeMoreButton onClick={() => setVisible((prev) => !prev)}>
        {isVisible ? 'See Less' : 'See More'}
      </SeeMoreButton>
      {isVisible && <Explanation>{text}</Explanation>}
    </div>
  );
};

// Component to render labels
const LabelsSection: React.FC<{ title: string; labels: any }> = ({
  title,
  labels,
}) => (
  <DetailsSection>
    <KeyvalText>
      <strong>{title}:</strong>
    </KeyvalText>
    {Object.entries(labels).map(([key, label]: [string, any]) => (
      <Section key={key}>
        <KeyvalText>
          {label.name}: {String(label.value)}
        </KeyvalText>
        <ToggleExplanation text={label.explain} />
      </Section>
    ))}
  </DetailsSection>
);

// Component to render instrumentation config
const InstrumentationSection: React.FC<{ title: string; config: any }> = ({
  title,
  config,
}) => (
  <DetailsSection>
    <KeyvalText>
      <strong>{title}:</strong>
    </KeyvalText>
    {Object.entries(config).map(([key, item]: [string, any]) => (
      <Section key={key}>
        <KeyvalText>
          {item.name}: {item.value}
        </KeyvalText>
        <ToggleExplanation text={item.explain} />
      </Section>
    ))}
  </DetailsSection>
);

// Component to render runtime info
const RuntimeInfoSection: React.FC<{ runtimeInfo: any }> = ({
  runtimeInfo,
}) => (
  <DetailsSection>
    <KeyvalText>
      <strong>Runtime Info:</strong>
    </KeyvalText>
    {runtimeInfo.containers.map((container: any, index: number) => (
      <ContainerSection key={index} container={container} />
    ))}
  </DetailsSection>
);

// Component for each container in runtime info
const ContainerSection: React.FC<{ container: any }> = ({ container }) => (
  <DetailsSection>
    <KeyvalText>
      <strong>Container Name:</strong> {container.containerName.value}
    </KeyvalText>
    <ToggleExplanation text={container.containerName.explain} />
    <KeyvalText>
      <strong>Language:</strong> {container.language?.value}
    </KeyvalText>
    <ToggleExplanation text={container.language?.explain} />
    <KeyvalText>
      <strong>Runtime Version:</strong> {container.runtimeVersion?.value}
    </KeyvalText>
    <ToggleExplanation text={container.runtimeVersion?.explain} />
  </DetailsSection>
);

// Component to render pods section
const PodsSection: React.FC<{
  pods: any[];
  podsPhasesCount: string;
  totalPods: number;
}> = ({ pods, podsPhasesCount, totalPods }) => (
  <DetailsSection>
    <KeyvalText>
      <strong>Pods:</strong>
    </KeyvalText>
    <KeyvalText>Total Pods: {totalPods}</KeyvalText>
    <KeyvalText>Phases Count: {podsPhasesCount}</KeyvalText>
    {pods.map((pod, index) => (
      <PodSection key={index} pod={pod} />
    ))}
  </DetailsSection>
);

// Component for each pod
const PodSection: React.FC<{ pod: any }> = ({ pod }) => (
  <DetailsSection>
    <KeyvalText>
      <strong>Pod Name:</strong> {pod.podName.value}
    </KeyvalText>
    <ToggleExplanation text={pod.podName.explain} />
    <KeyvalText>
      <strong>Node Name:</strong> {pod.nodeName.value}
    </KeyvalText>
    <ToggleExplanation text={pod.nodeName.explain} />
    <KeyvalText>
      <strong>Phase:</strong> {pod.phase.value}
    </KeyvalText>
    <ToggleExplanation text={pod.phase.explain} />
    {pod.containers.map((container: any, index: number) => (
      <ContainerSection key={index} container={container} />
    ))}
  </DetailsSection>
);

const Explanation = styled.p`
  font-size: 0.85rem;
  color: ${theme.colors.light_grey};
  margin-bottom: 10px;
`;

const Section = styled.div`
  margin-bottom: 20px;
`;

const IconWrapper = styled.div`
  position: relative;
  padding: 8px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  cursor: pointer;
  &:hover {
    background-color: ${theme.colors.light_grey};
  }
`;

const LoadingMessage = styled.p`
  font-size: 1rem;
  color: ${theme.colors.light_grey};
`;

const DescriptionContent = styled(KeyvalText)`
  white-space: pre-wrap;
  line-height: 1.6;
  padding: 20px;
`;

const DetailsSection = styled.div`
  margin-top: 16px;
`;

const SeeMoreButton = styled.button``;
