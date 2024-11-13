import React from 'react';
import { type UseSourceFormDataResponse } from '@/hooks';
import { ChooseSourcesBodyFast } from './choose-sources-body-fast';
import { ChooseSourcesBodySimple } from './choose-sources-body-simple';

interface Props extends UseSourceFormDataResponse {
  componentType: 'SIMPLE' | 'FAST';
  isModal?: boolean;
}

export const ChooseSourcesBody: React.FC<Props> = ({ componentType, ...props }) => {
  switch (componentType) {
    case 'SIMPLE':
      return <ChooseSourcesBodySimple {...props} />;

    case 'FAST':
      return <ChooseSourcesBodyFast {...props} />;

    default:
      return null;
  }
};
