import React from 'react';
import PiiMasking from './pii-masking';
import ErrorSampler from './error-sampler';
import LatencySampler from './latency-sampler';
import AddClusterInfo from './add-cluster-info';
import DeleteAttributes from './delete-attributes';
import RenameAttributes from './rename-attributes';
import { ACTION_TYPE } from '@odigos/ui-components';
import ProbabilisticSampler from './probabilistic-sampler';

interface Props {
  actionType?: ACTION_TYPE;
  value: string;
  setValue: (value: string) => void;
  errorMessage?: string;
}

interface ComponentProps {
  value: Props['value'];
  setValue: Props['setValue'];
  errorMessage?: Props['errorMessage'];
}

type ComponentType = React.FC<ComponentProps> | null;

const componentsMap: Record<ACTION_TYPE, ComponentType> = {
  [ACTION_TYPE.ADD_CLUSTER_INFO]: AddClusterInfo,
  [ACTION_TYPE.DELETE_ATTRIBUTES]: DeleteAttributes,
  [ACTION_TYPE.RENAME_ATTRIBUTES]: RenameAttributes,
  [ACTION_TYPE.PII_MASKING]: PiiMasking,
  [ACTION_TYPE.ERROR_SAMPLER]: ErrorSampler,
  [ACTION_TYPE.PROBABILISTIC_SAMPLER]: ProbabilisticSampler,
  [ACTION_TYPE.LATENCY_SAMPLER]: LatencySampler,
};

const ActionCustomFields: React.FC<Props> = ({ actionType, value, setValue, errorMessage }) => {
  if (!actionType) return null;

  const Component = componentsMap[actionType];

  return Component ? <Component value={value} setValue={setValue} errorMessage={errorMessage} /> : null;
};

export default ActionCustomFields;
