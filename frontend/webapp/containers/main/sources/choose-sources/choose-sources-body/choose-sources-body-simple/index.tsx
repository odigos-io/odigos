import React from 'react';
import { ModalBody } from '@/styles';
import { SourcesList } from './sources-list';
import { SourceControls } from './source-controls';
import { type UseSourceFormDataResponse } from '@/hooks';

interface Props extends UseSourceFormDataResponse {
  isModal?: boolean;
}

export const ChooseSourcesBodySimple: React.FC<Props> = (props) => {
  return (
    <ModalBody $isModal={props.isModal}>
      <SourceControls {...props} />
      <SourcesList {...props} />
    </ModalBody>
  );
};
