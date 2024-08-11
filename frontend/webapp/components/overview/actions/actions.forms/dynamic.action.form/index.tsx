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

interface DynamicActionFormProps<T = any> {
  type?: string;
  data: T;
  onChange: (key: string, value: T | null) => void;
  setIsFormValid?: (isValid: boolean) => void;
}

export function DynamicActionForm({
  type,
  data,
  onChange,
  setIsFormValid = () => {},
}: DynamicActionFormProps): React.JSX.Element {
  const formComponents = {
    [ActionsType.ADD_CLUSTER_INFO]: AddClusterInfoForm,
    [ActionsType.DELETE_ATTRIBUTES]: DeleteAttributesForm,
    [ActionsType.RENAME_ATTRIBUTES]: RenameAttributesForm,
    [ActionsType.ERROR_SAMPLER]: ErrorSamplerForm,
    [ActionsType.PROBABILISTIC_SAMPLER]: ProbabilisticSamplerForm,
    [ActionsType.LATENCY_SAMPLER]: LatencySamplerForm,
    [ActionsType.PII_MASKING]: PiiMaskingForm,
  };

  const FormComponent = type ? formComponents[type] : null;

  return (
    <>
      {FormComponent ? (
        <FormComponent

          data={data}
          onChange={onChange}
          setIsFormValid={setIsFormValid}
        />
      ) : (
        <div>No action form available</div>
      )}
    </>
  );
}
