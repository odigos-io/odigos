import React from 'react';
import { WarningModal } from '@/reuseable-components';

interface Props {
  isOpen: boolean;
  name?: string;
  onApprove: () => void;
  onDeny: () => void;
}

const DeleteWarning: React.FC<Props> = ({ isOpen, name, onApprove, onDeny }) => {
  return (
    <WarningModal
      isOpen={isOpen}
      noOverlay
      title={`Delete${name ? ` ${name}` : ''}`}
      description='Are you sure you want to delete this?'
      approveButton={{
        text: 'Delete',
        variant: 'danger',
        onClick: onApprove,
      }}
      denyButton={{
        text: 'Cancel',
        onClick: onDeny,
      }}
    />
  );
};

export { DeleteWarning };
