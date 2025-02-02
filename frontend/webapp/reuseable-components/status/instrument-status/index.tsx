import React from 'react';
import { Status, type StatusProps } from '@/reuseable-components';
import { INSTUMENTATION_STATUS, WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';

interface Props extends StatusProps {
  language: WORKLOAD_PROGRAMMING_LANGUAGES;
}

export const InstrumentStatus: React.FC<Props> = ({ language, ...props }) => {
  const isActive = ![
    WORKLOAD_PROGRAMMING_LANGUAGES.IGNORED,
    WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN,
    WORKLOAD_PROGRAMMING_LANGUAGES.PROCESSING,
    WORKLOAD_PROGRAMMING_LANGUAGES.NO_CONTAINERS,
    WORKLOAD_PROGRAMMING_LANGUAGES.NO_RUNNING_PODS,
  ].includes(language);

  return <Status title={isActive ? INSTUMENTATION_STATUS.INSTRUMENTED : INSTUMENTATION_STATUS.UNINSTRUMENTED} isActive={isActive} withIcon withBorder {...props} />;
};
