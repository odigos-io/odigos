import React from 'react';
import { ActionsType } from '@/types';
import AddClusterInfo from './add-cluster-info';
import DeleteAttributes from './delete-attributes';
import RenameAttributes from './rename-attributes';
import PiiMasking from './pii-masking';
import ErrorSampler from './error-sampler';
import ProbabilisticSampler from './probabilistic-sampler';
import LatencySampler from './latency-sampler';

interface ActionCustomFieldsProps {
  actionType?: ActionsType;
  value: string;
  setValue: (value: string) => void;
}

type ComponentProps = {
  value: string;
  setValue: (value: string) => void;
};

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

const ActionCustomFields: React.FC<ActionCustomFieldsProps> = ({ actionType, value, setValue }) => {
  if (!actionType) return null;

  const Component = componentsMap[actionType];

  return Component ? <Component value={value} setValue={setValue} /> : null;
};

export default ActionCustomFields;
