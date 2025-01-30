import React from 'react';
import PiiMasking from './pii-masking';
import ErrorSampler from './error-sampler';
import LatencySampler from './latency-sampler';
import AddClusterInfo from './add-cluster-info';
import DeleteAttributes from './delete-attributes';
import RenameAttributes from './rename-attributes';
import ProbabilisticSampler from './probabilistic-sampler';
import { Types } from '@odigos/ui-components';

interface Props {
  actionType?: Types.ACTION_TYPE;
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

const componentsMap: Record<Types.ACTION_TYPE, ComponentType> = {
  [Types.ACTION_TYPE.ADD_CLUSTER_INFO]: AddClusterInfo,
  [Types.ACTION_TYPE.DELETE_ATTRIBUTES]: DeleteAttributes,
  [Types.ACTION_TYPE.RENAME_ATTRIBUTES]: RenameAttributes,
  [Types.ACTION_TYPE.PII_MASKING]: PiiMasking,
  [Types.ACTION_TYPE.ERROR_SAMPLER]: ErrorSampler,
  [Types.ACTION_TYPE.PROBABILISTIC_SAMPLER]: ProbabilisticSampler,
  [Types.ACTION_TYPE.LATENCY_SAMPLER]: LatencySampler,
};

const ActionCustomFields: React.FC<Props> = ({ actionType, value, setValue, errorMessage }) => {
  if (!actionType) return null;

  const Component = componentsMap[actionType];

  return Component ? <Component value={value} setValue={setValue} errorMessage={errorMessage} /> : null;
};

export default ActionCustomFields;
