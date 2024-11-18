import React from 'react';
import { WarningModal } from '@/reuseable-components';
import { NotificationType } from '@/types';

interface Props {
  isOpen: boolean;
  noOverlay?: boolean;
  name?: string;
  note?: {
    type: NotificationType;
    title: string;
    message: string;
  };
  onApprove: () => void;
  onDeny: () => void;
}

const DeleteWarning: React.FC<Props> = ({ isOpen, noOverlay, name, note, onApprove, onDeny }) => {
  return (
    <WarningModal
      isOpen={isOpen}
      noOverlay={noOverlay}
      title={`Delete${name ? ` ${name}` : ''}`}
      description='Are you sure you want to delete?'
      note={note}
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
