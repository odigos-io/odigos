import { PropsWithChildren, useRef, useState } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import DrawerHeader, { DrawerHeaderRef } from './drawer-header';
import DrawerFooter from './drawer-footer';
import { Drawer, WarningModal } from '@/reuseable-components';

const DRAWER_WIDTH = '640px';

interface Props {
  title: string;
  imageUri: string;
  isEdit: boolean;
  clickEdit: (bool?: boolean) => void;
  clickSave: (newTitle: string) => void;
  clickDelete: () => void;
  clickCancel: () => void;
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

const OverviewDrawer: React.FC<Props & PropsWithChildren> = ({
  children,
  title,
  imageUri,
  isEdit,
  clickEdit,
  clickSave,
  clickDelete,
  clickCancel,
}) => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const setSelectedItem = useDrawerStore(({ setSelectedItem }) => setSelectedItem);

  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isCancelModalOpen, setIsCancelModalOpen] = useState(false);

  const titleRef = useRef<DrawerHeaderRef>(null);

  const closeDrawer = () => {
    setSelectedItem(null);
    clickEdit(false);
    setIsDeleteModalOpen(false);
    setIsCancelModalOpen(false);
  };

  const closeWarningModals = () => {
    setIsDeleteModalOpen(false);
    setIsCancelModalOpen(false);
  };

  const handleCancel = () => setIsCancelModalOpen(true);
  const handleDelete = () => setIsDeleteModalOpen(true);

  return (
    <>
      <Drawer isOpen onClose={closeDrawer} width={DRAWER_WIDTH} closeOnEscape={!isDeleteModalOpen}>
        <DrawerContent>
          <DrawerHeader
            ref={titleRef}
            title={title}
            imageUri={imageUri}
            isEdit={isEdit}
            onEdit={() => clickEdit(true)}
            onClose={isEdit ? handleCancel : closeDrawer}
          />

          <ContentArea>{children}</ContentArea>

          {isEdit && <DrawerFooter onSave={() => clickSave(titleRef.current?.getTitle() || '')} onCancel={handleCancel} onDelete={handleDelete} />}
        </DrawerContent>
      </Drawer>

      <WarningModal
        isOpen={isDeleteModalOpen}
        noOverlay
        title={`Delete ${title}`}
        description={`Are you sure you want to delete this ${selectedItem?.type}?`}
        approveButton={{
          text: 'Delete',
          variant: 'danger',
          onClick: () => {
            clickDelete();
            closeWarningModals();
          },
        }}
        denyButton={{
          text: 'Cancel',
          onClick: closeWarningModals,
        }}
      />

      <WarningModal
        isOpen={isCancelModalOpen}
        noOverlay
        title='Cancel edit mode'
        description='Are you sure you want to cancel?'
        approveButton={{
          text: 'Cancel',
          variant: 'warning',
          onClick: () => {
            titleRef.current?.clearTitle();
            clickCancel();
            closeWarningModals();
          },
        }}
        denyButton={{
          text: 'Go Back',
          onClick: closeWarningModals,
        }}
      />
    </>
  );
};

export default OverviewDrawer;
