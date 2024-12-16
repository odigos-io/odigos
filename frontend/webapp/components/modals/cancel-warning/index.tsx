import React from 'react';
import { NOTIFICATION_TYPE } from '@/types';
import { WarningModal } from '@/reuseable-components';

interface Props {
  isOpen: boolean;
  noOverlay?: boolean;
  name?: string;
  onApprove: () => void;
  onDeny: () => void;
}

const CancelWarning: React.FC<Props> = ({ isOpen, noOverlay, name, onApprove, onDeny }) => {
  return (
    <WarningModal
      isOpen={isOpen}
      noOverlay={noOverlay}
      title={`Cancel${name ? ` ${name}` : ''}`}
      description='Are you sure you want to cancel?'
      approveButton={{
        text: 'Confirm',
        variant: NOTIFICATION_TYPE.WARNING,
        onClick: onApprove,
      }}
      denyButton={{
        text: 'Go Back',
        onClick: onDeny,
      }}
    />
  );
};

export { CancelWarning };
