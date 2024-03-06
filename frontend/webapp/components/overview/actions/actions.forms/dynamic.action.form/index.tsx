'use client';
import React from 'react';
import { ActionsType } from '@/types';
import { AddClusterInfoForm } from '../add.cluster.info';

interface DynamicActionFormProps {
  type: string | undefined;
  data: any;
  onChange: (key: string, value: any) => void;
}

export function DynamicActionForm({
  type,
  data,
  onChange,
}: DynamicActionFormProps): React.JSX.Element {
  function renderCurrentAction() {
    switch (type) {
      case ActionsType.ADD_CLUSTER_INFO:
        return <AddClusterInfoForm data={data} onChange={onChange} />;
      default:
        return <div></div>;
    }
  }

  return <>{renderCurrentAction()}</>;
}
