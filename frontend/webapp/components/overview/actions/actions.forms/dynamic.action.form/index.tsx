'use client';
import React from 'react';
import { ActionsType } from '@/types';
import { AddClusterInfoForm } from '../add.cluster.info';
import { DeleteAttributesForm } from '../delete.attribute';
import { RenameAttributesForm } from '../rename.attributes';
import {
  ErrorSamplerForm,
  LatencySamplerForm,
  ProbabilisticSamplerForm,
} from '../samplers';
import { PiiMaskingForm } from '../pii-masking';

interface DynamicActionFormProps {
  type: string | undefined;
  data: any;
  onChange: (key: string, value: any) => void;
  setIsFormValid?: (isValid: boolean) => void;
}

export function DynamicActionForm({
  type,
  data,
  onChange,
  setIsFormValid,
}: DynamicActionFormProps): React.JSX.Element {
  function renderCurrentAction() {
    switch (type) {
      case ActionsType.ADD_CLUSTER_INFO:
        return <AddClusterInfoForm data={data} onChange={onChange} />;
      case ActionsType.DELETE_ATTRIBUTES:
        return <DeleteAttributesForm data={data} onChange={onChange} />;
      case ActionsType.RENAME_ATTRIBUTES:
        return <RenameAttributesForm data={data} onChange={onChange} />;
      case ActionsType.ERROR_SAMPLER:
        return <ErrorSamplerForm data={data} onChange={onChange} />;
      case ActionsType.PROBABILISTIC_SAMPLER:
        return <ProbabilisticSamplerForm data={data} onChange={onChange} />;
      case ActionsType.LATENCY_SAMPLER:
        return (
          <LatencySamplerForm
            data={data}
            onChange={onChange}
            setIsFormValid={setIsFormValid}
          />
        );
      case ActionsType.PII_MASKING:
        return (
          <PiiMaskingForm
            data={data}
            onChange={onChange}
            setIsFormValid={setIsFormValid}
          />
        );
      default:
        return <div></div>;
    }
  }

  return <>{renderCurrentAction()}</>;
}
