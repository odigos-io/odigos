import React from 'react';
import { ModalBody } from '@/styles';
import { SourcesList } from './sources-list';
import { SourceControls } from './source-controls';
import { type UseSourceFormDataResponse } from '@/hooks';

interface Props extends UseSourceFormDataResponse {
  isModal?: boolean;
}

export const ChooseSourcesBodyFast: React.FC<Props> = (props) => {
  return (
    <ModalBody $isNotModal={!props.isModal}>
      <SourceControls {...props} />
      <SourcesList {...props} />
    </ModalBody>
  );
};
