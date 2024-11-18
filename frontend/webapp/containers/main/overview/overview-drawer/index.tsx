import { PropsWithChildren, useRef, useState } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import DrawerFooter from './drawer-footer';
import { Drawer } from '@/reuseable-components';
import DrawerHeader, { DrawerHeaderRef } from './drawer-header';
import { CancelWarning, DeleteWarning } from '@/components/modals';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { useDestinationCRUD, useSourceCRUD } from '@/hooks';

// + 64 because of padding
const DRAWER_WIDTH = `${640 + 64}px`;

interface Props {
  title: string;
  imageUri: string;
  isEdit: boolean;
  isFormDirty: boolean;
  onEdit: (bool?: boolean) => void;
  onSave: (newTitle: string) => void;
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

  const titleRef = useRef<DrawerHeaderRef>(null);

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
    titleRef.current?.clearTitle();
    onCancel();
    closeWarningModals();
  };

  const clickCancel = () => {
    const isTitleDirty = titleRef.current?.getTitle() !== title;
    if (isFormDirty || isTitleDirty) {
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
    onSave(titleRef.current?.getTitle() || '');
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
          <DrawerHeader ref={titleRef} title={title} imageUri={imageUri} isEdit={isEdit} onEdit={() => onEdit(true)} onClose={isEdit ? clickCancel : closeDrawer} />
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
