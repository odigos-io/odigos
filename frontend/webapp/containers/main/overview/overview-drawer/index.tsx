import { PropsWithChildren, useRef, useState } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import DrawerFooter from './drawer-footer';
import { Drawer } from '@/reuseable-components';
import DrawerHeader, { DrawerHeaderRef } from './drawer-header';
import { CancelWarning, DeleteWarning } from '@/components/modals';

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

      <DeleteWarning
        isOpen={isDeleteModalOpen}
        name={`${selectedItem?.type}${title ? ` (${title})` : ''}`}
        onApprove={() => {
          clickDelete();
          closeWarningModals();
        }}
        onDeny={closeWarningModals}
      />

      <CancelWarning
        isOpen={isCancelModalOpen}
        name='edit mode'
        onApprove={() => {
          titleRef.current?.clearTitle();
          clickCancel();
          closeWarningModals();
        }}
        onDeny={closeWarningModals}
      />
    </>
  );
};

export default OverviewDrawer;
