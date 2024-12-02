import React from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { Text } from '@/reuseable-components';
import { getStatusIcon, INSTUMENTATION_STATUS, WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';

interface Props {
  language: WORKLOAD_PROGRAMMING_LANGUAGES;
}

const Container = styled.div<{ $active: boolean }>`
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  margin-left: auto;
  border-radius: 360px;
  border: 1px solid ${({ $active, theme }) => ($active ? theme.colors.dark_green : theme.colors.border)};
`;

export const InstrumentStatus: React.FC<Props> = ({ language }) => {
  const active = ![
    WORKLOAD_PROGRAMMING_LANGUAGES.IGNORED,
    WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN,
    WORKLOAD_PROGRAMMING_LANGUAGES.PROCESSING,
    WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS,
    WORKLOAD_PROGRAMMING_LANGUAGES.NO_RUNNING_PODS,
  ].includes(language);

  return (
    <Container $active={active}>
      <Image src={active ? getStatusIcon('success') : '/icons/common/circled-cross.svg'} alt='' width={12} height={12} />
      <Text size={12} family='secondary' color={active ? theme.text.success : theme.text.grey}>
        {active ? INSTUMENTATION_STATUS.INSTRUMENTED : INSTUMENTATION_STATUS.UNINSTRUMENTED}
      </Text>
    </Container>
  );
};
