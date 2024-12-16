import { PropsWithChildren, useRef, useState } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import DrawerFooter from './drawer-footer';
import { Drawer } from '@/reuseable-components';
import { OVERVIEW_ENTITY_TYPES } from '@/types';
import DrawerHeader, { DrawerHeaderRef } from './drawer-header';
import { CancelWarning, DeleteWarning } from '@/components/modals';
import { useDestinationCRUD, useKeyDown, useSourceCRUD } from '@/hooks';

const DRAWER_WIDTH = `${640 + 64}px`; // +64 because of "ContentArea" padding

interface Props {
  title: string;
  titleTooltip?: string;
  imageUri?: string;
  isEdit?: boolean;
  isFormDirty?: boolean;
  onEdit?: (bool?: boolean) => void;
  onSave?: (newTitle: string) => void;
  onDelete?: () => void;
  onCancel?: () => void;
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

const OverviewDrawer: React.FC<Props & PropsWithChildren> = ({ children, title, titleTooltip, imageUri, isEdit = false, isFormDirty = false, onEdit, onSave, onDelete, onCancel }) => {
  const { selectedItem, setSelectedItem } = useDrawerStore();

  useKeyDown({ key: 'Enter', active: !!selectedItem }, () => (isEdit ? clickSave() : closeDrawer()));

  const { sources } = useSourceCRUD();
  const { destinations } = useDestinationCRUD();

  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isCancelModalOpen, setIsCancelModalOpen] = useState(false);

  const titleRef = useRef<DrawerHeaderRef>(null);

  const isSource = selectedItem?.type === OVERVIEW_ENTITY_TYPES.SOURCE;
  const isDestination = selectedItem?.type === OVERVIEW_ENTITY_TYPES.DESTINATION;

  const closeDrawer = () => {
    setSelectedItem(null);
    if (onEdit) onEdit(false);
    setIsDeleteModalOpen(false);
    setIsCancelModalOpen(false);
  };

  const closeWarningModals = () => {
    setIsDeleteModalOpen(false);
    setIsCancelModalOpen(false);
  };

  const handleCancel = () => {
    titleRef.current?.clearTitle();
    if (onCancel) onCancel();
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
    if (onDelete) onDelete();
    closeWarningModals();
  };

  const clickDelete = () => {
    setIsDeleteModalOpen(true);
  };

  const clickSave = () => {
    if (onSave) onSave(titleRef.current?.getTitle() || '');
  };

  const isLastItem = () => {
    let isLast = false;

    if (isSource) isLast = sources.length === 1;
    if (isDestination) isLast = destinations.length === 1;

    return isLast;
  };

  return (
    <>
      <Drawer isOpen onClose={isEdit ? clickCancel : closeDrawer} width={DRAWER_WIDTH} closeOnEscape={!isDeleteModalOpen && !isCancelModalOpen}>
        <DrawerContent>
          <DrawerHeader
            ref={titleRef}
            title={title}
            titleTooltip={titleTooltip}
            imageUri={imageUri}
            isEdit={isEdit}
            onEdit={onEdit ? () => onEdit(true) : undefined}
            onClose={isEdit ? clickCancel : closeDrawer}
          />
          <ContentArea>{children}</ContentArea>
          <DrawerFooter isOpen={isEdit} onSave={clickSave} onCancel={clickCancel} onDelete={clickDelete} deleteLabel={isSource ? 'Uninstrument' : undefined} />
        </DrawerContent>
      </Drawer>

      <DeleteWarning
        isOpen={isDeleteModalOpen}
        noOverlay
        name={`${selectedItem?.type}${title ? ` (${title})` : ''}`}
        type={selectedItem?.type as OVERVIEW_ENTITY_TYPES}
        isLastItem={isLastItem()}
        onApprove={handleDelete}
        onDeny={closeWarningModals}
      />
      <CancelWarning isOpen={isCancelModalOpen} noOverlay name='edit mode' onApprove={handleCancel} onDeny={closeWarningModals} />
    </>
  );
};

export default OverviewDrawer;
