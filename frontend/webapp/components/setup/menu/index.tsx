import React, { useEffect } from 'react';
import { CheckIcon } from '@/assets';
import { type StepProps } from '@/types';
import { Text } from '@/reuseable-components';
import styled, { css } from 'styled-components';

const Container = styled.div`
  display: flex;
  flex-direction: column;
  gap: 32px;
`;

const Step = styled.div<{ $state: 'finish' | 'active' | 'disabled' }>`
  display: flex;
  gap: 16px;
  padding: 8px;
  opacity: ${({ $state }) => ($state === 'disabled' ? 0.5 : 1)};
  transition: all 0.3s;

  ${({ $state }) =>
    $state === 'finish' &&
    css`
      opacity: 0.8;
    `}

  ${({ $state }) => $state === 'active' && css``}

  ${({ $state }) =>
    $state === 'disabled' &&
    css`
      opacity: 0.5;
    `}

  & + & {
    margin-top: 10px;
  }
`;

const IconWrapper = styled.div<{ $state: 'finish' | 'active' | 'disabled' }>`
  border-radius: 32px;
  width: 24px;
  height: 24px;
  border: 1px solid ${({ theme }) => theme.colors.secondary};
  display: flex;
  justify-content: center;
  align-items: center;

  ${({ $state }) =>
    $state === 'finish'
      ? css`
          opacity: 0.8;
        `
      : $state === 'disabled' &&
        css`
          border: 1px dashed ${({ theme }) => theme.colors.white_opacity['40']};
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

export const SideMenu: React.FC<{ data?: StepProps[]; currentStep?: number }> = ({ data, currentStep }) => {
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

      setStepsList(currentSteps);
    }
  }, [currentStep, data]);

  return (
    <Container>
      {stepsList.map((step, index) => (
        <Step key={index} $state={step.state}>
          <IconWrapper $state={step.state}>
            {step.state === 'finish' && <CheckIcon size={20} />}
            {step.state === 'active' && <Text size={12}>{step.stepNumber}</Text>}
            {step.state === 'disabled' && <Text size={12}>{step.stepNumber}</Text>}
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
