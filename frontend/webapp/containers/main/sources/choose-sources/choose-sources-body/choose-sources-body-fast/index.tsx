import React from 'react';
import { ModalBody } from '@/styles';
import { UseConnectSourcesMenuStateResponse } from '@/hooks';

interface Props extends UseConnectSourcesMenuStateResponse {
  isModal?: boolean;
}

export const ChooseSourcesBodyFast: React.FC<Props> = ({ isModal = false }) => {
  return <ModalBody></ModalBody>;
};
