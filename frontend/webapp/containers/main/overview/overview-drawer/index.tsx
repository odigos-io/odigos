import { PropsWithChildren, useState } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import DrawerFooter from './drawer-footer';
import DrawerHeader from './drawer-header';
import { Drawer } from '@/reuseable-components';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { useDestinationCRUD, useSourceCRUD } from '@/hooks';
import { CancelWarning, DeleteWarning } from '@/components/modals';

const DRAWER_WIDTH = `${640 + 64}px`; // +64 because of "ContentArea" padding

interface Props {
  title: string;
  imageUri: string;
  isEdit: boolean;
  isFormDirty: boolean;
  onEdit: (bool?: boolean) => void;
  onSave: () => void;
  onDelete: () => void;
  onCancel: () => void;
}

const DrawerContent = styled.div`
  display: flex;
  flex-direction: column;
  height: 100%;
`;

const ContentArea = styled.div`
  flex-grow: 1;
  padding: 24px 32px;
  overflow-y: auto;
`;

const OverviewDrawer: React.FC<Props & PropsWithChildren> = ({ children, title, imageUri, isEdit, isFormDirty, onEdit, onSave, onDelete, onCancel }) => {
  const { sources } = useSourceCRUD();
  const { destinations } = useDestinationCRUD();
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const setSelectedItem = useDrawerStore(({ setSelectedItem }) => setSelectedItem);

  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isCancelModalOpen, setIsCancelModalOpen] = useState(false);

  const closeDrawer = () => {
    setSelectedItem(null);
    onEdit(false);
    setIsDeleteModalOpen(false);
    setIsCancelModalOpen(false);
  };

  const closeWarningModals = () => {
    setIsDeleteModalOpen(false);
    setIsCancelModalOpen(false);
  };

  const handleCancel = () => {
    onCancel();
    closeWarningModals();
  };

  const clickCancel = () => {
    if (isFormDirty) {
      setIsCancelModalOpen(true);
    } else {
      handleCancel();
    }
  };

  const handleDelete = () => {
    onDelete();
    closeWarningModals();
  };

  const clickDelete = () => {
    setIsDeleteModalOpen(true);
  };

  const clickSave = () => {
    onSave();
  };

  const isLastItem = () => {
    let isLast = false;

    if (selectedItem?.type === OVERVIEW_ENTITY_TYPES.SOURCE) isLast = sources.length === 1;
    if (selectedItem?.type === OVERVIEW_ENTITY_TYPES.DESTINATION) isLast = destinations.length === 1;

    return isLast;
  };

  return (
    <>
      <Drawer isOpen onClose={isEdit ? clickCancel : closeDrawer} width={DRAWER_WIDTH} closeOnEscape={!isDeleteModalOpen && !isCancelModalOpen}>
        <DrawerContent>
          <DrawerHeader title={title} imageUri={imageUri} isEdit={isEdit} onEdit={() => onEdit(true)} onClose={isEdit ? clickCancel : closeDrawer} />
          <ContentArea>{children}</ContentArea>
          {isEdit && <DrawerFooter onSave={clickSave} onCancel={clickCancel} onDelete={clickDelete} />}
        </DrawerContent>
      </Drawer>

      <DeleteWarning
        isOpen={isDeleteModalOpen}
        noOverlay
        name={`${selectedItem?.type}${title ? ` (${title})` : ''}`}
        note={
          isLastItem()
            ? {
                type: 'warning',
                title: `You're about to delete the last ${selectedItem?.type}`,
                message: 'This will break your pipeline!',
              }
            : undefined
        }
        onApprove={handleDelete}
        onDeny={closeWarningModals}
      />
      <CancelWarning isOpen={isCancelModalOpen} noOverlay name='edit mode' onApprove={handleCancel} onDeny={closeWarningModals} />
    </>
  );
};

export default OverviewDrawer;
