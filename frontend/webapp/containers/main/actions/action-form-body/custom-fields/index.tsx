import React from 'react';
import { ActionsType } from '@/types';
import PiiMasking from './pii-masking';
import ErrorSampler from './error-sampler';
import LatencySampler from './latency-sampler';
import AddClusterInfo from './add-cluster-info';
import DeleteAttributes from './delete-attributes';
import RenameAttributes from './rename-attributes';
import ProbabilisticSampler from './probabilistic-sampler';

interface Props {
  actionType?: ActionsType;
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

const componentsMap: Record<ActionsType, ComponentType> = {
  [ActionsType.ADD_CLUSTER_INFO]: AddClusterInfo,
  [ActionsType.DELETE_ATTRIBUTES]: DeleteAttributes,
  [ActionsType.RENAME_ATTRIBUTES]: RenameAttributes,
  [ActionsType.PII_MASKING]: PiiMasking,
  [ActionsType.ERROR_SAMPLER]: ErrorSampler,
  [ActionsType.PROBABILISTIC_SAMPLER]: ProbabilisticSampler,
  [ActionsType.LATENCY_SAMPLER]: LatencySampler,
};

const ActionCustomFields: React.FC<Props> = ({ actionType, value, setValue, errorMessage }) => {
  if (!actionType) return null;

  const Component = componentsMap[actionType];

  return Component ? <Component value={value} setValue={setValue} errorMessage={errorMessage} /> : null;
};

export default ActionCustomFields;
