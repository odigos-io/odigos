import React from 'react';
import { WarningModal } from '@/reuseable-components';

interface Props {
  isOpen: boolean;
  noOverlay?: boolean;
  name?: string;
  warnAgain?: {
    title: string;
    description: string;
  };
  onApprove: () => void;
  onDeny: () => void;
}

const DeleteWarning: React.FC<Props> = ({ isOpen, noOverlay, name, warnAgain, onApprove, onDeny }) => {
  return (
    <WarningModal
      isOpen={isOpen}
      noOverlay={noOverlay}
      title={`Delete${name ? ` ${name}` : ''}`}
      description='Are you sure you want to delete?'
      warnAgain={warnAgain}
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
