import Image from 'next/image';
import { StepProps } from '@/types';
import React, { useEffect } from 'react';
import { Text } from '@/reuseable-components';
import styled, { css } from 'styled-components';

const Container = styled.div`
  display: flex;
  flex-direction: column;
  gap: 32px;
`;

const Step = styled.div<{ state: 'finish' | 'active' | 'disabled' }>`
  display: flex;
  gap: 16px;
  padding: 10px 0;
  cursor: ${({ state }) => (state === 'disabled' ? 'auto' : 'auto')};
  opacity: ${({ state }) => (state === 'disabled' ? 0.5 : 1)};

  transition: opacity 0.3s;

  ${({ state }) => state === 'active' && css``}

  & + & {
    margin-top: 10px;
  }
`;

const IconWrapper = styled.div<{ state: 'finish' | 'active' | 'disabled' }>`
  border-radius: 32px;
  width: 24px;
  height: 24px;
  border: 1px solid ${({ theme }) => theme.colors.secondary};
  display: flex;
  justify-content: center;
  align-items: center;

  ${({ state }) =>
    state === 'finish'
      ? css`
          opacity: 0.8;
        `
      : state === 'disabled' &&
        css`
          border: 1px dashed rgba(249, 249, 249, 0.4);
        `}
`;

const StepContent = styled.div`
  display: flex;
  justify-content: center;
  flex-direction: column;
  gap: 8px;
`;

const StepTitle = styled(Text)`
  font-weight: 500;
`;

const StepSubtitle = styled(Text)``;

const SideMenu: React.FC<{ data?: StepProps[]; currentStep?: number }> = ({
  data,
  currentStep,
}) => {
  const [stepsList, setStepsList] = React.useState<StepProps[]>([]);
  const steps: StepProps[] = data || [
    {
      title: 'INSTALLATION',
      subtitle: 'Success',
      state: 'finish',
      stepNumber: 1,
    },
    {
      title: 'SOURCES',
      state: 'active',
      subtitle: '',

      stepNumber: 2,
    },
    {
      title: 'DESTINATIONS',
      state: 'disabled',
      stepNumber: 3,
    },
  ];
  useEffect(() => {
    console.log({ currentStep });
    if (currentStep) {
      const currentSteps = (data || steps).map((step, index) => {
        if (index < currentStep - 1) {
          return { ...step, state: 'finish' as const };
        } else if (index === currentStep - 1) {
          return { ...step, state: 'active' as const };
        } else {
          return { ...step, state: 'disabled' as const };
        }
      });
      console.log({ currentSteps });
      setStepsList(currentSteps);
    }
  }, [currentStep, data]);

  return (
    <Container>
      {stepsList.map((step, index) => (
        <Step key={index} state={step.state}>
          <IconWrapper state={step.state}>
            {step.state === 'finish' && (
              <Image
                src="/icons/common/check.svg"
                width={20}
                height={20}
                alt={''}
              />
            )}
            {step.state === 'active' && (
              <Text size={12}>{step.stepNumber}</Text>
            )}
            {step.state === 'disabled' && (
              <Text size={12}>{step.stepNumber}</Text>
            )}
          </IconWrapper>
          <StepContent>
            <StepTitle family={'secondary'}>{step.title}</StepTitle>
            {step.subtitle && (
              <StepSubtitle size={10} weight={300}>
                {step.subtitle}
              </StepSubtitle>
            )}
          </StepContent>
        </Step>
      ))}
    </Container>
  );
};

export { SideMenu };
