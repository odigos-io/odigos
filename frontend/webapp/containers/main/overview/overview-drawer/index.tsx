import { PropsWithChildren, useMemo, useRef, useState } from 'react';
import { SVG } from '@/assets';
import styled from 'styled-components';
import DrawerFooter from './drawer-footer';
import { Drawer } from '@/reuseable-components';
import DrawerHeader, { DrawerHeaderRef } from './drawer-header';
import { CancelWarning, DeleteWarning } from '@/components/modals';
import { NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES } from '@/types';
import { useDestinationCRUD, useKeyDown, useSourceCRUD } from '@/hooks';
import { useDrawerStore, useNotificationStore, usePendingStore } from '@/store';

const DRAWER_WIDTH = `${640 + 64}px`; // +64 because of "ContentArea" padding

interface Props {
  title: string;
  titleTooltip?: string;
  icon?: SVG;
  iconSrc?: string;
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

const OverviewDrawer: React.FC<Props & PropsWithChildren> = ({ children, title, titleTooltip, icon, iconSrc, isEdit = false, isFormDirty = false, onEdit, onSave, onDelete, onCancel }) => {
  const { isThisPending } = usePendingStore();
  const { addNotification } = useNotificationStore();
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

  const isPending = useMemo(() => {
    if (!selectedItem?.type) return false;

    return isThisPending({
      entityType: selectedItem.type as OVERVIEW_ENTITY_TYPES,
      entityId: selectedItem.id,
    });
  }, [selectedItem]);

  const handlePending = (action: string) => {
    addNotification({
      type: NOTIFICATION_TYPE.WARNING,
      title: 'Pending',
      message: `Cannot click ${action}, ${selectedItem?.type} is pending`,
      hideFromHistory: true,
    });
  };

  return (
    <>
      <Drawer isOpen onClose={isEdit ? clickCancel : closeDrawer} width={DRAWER_WIDTH} closeOnEscape={!isDeleteModalOpen && !isCancelModalOpen}>
        <DrawerContent>
          <DrawerHeader
            ref={titleRef}
            title={title}
            titleTooltip={titleTooltip}
            icon={icon}
            iconSrc={iconSrc}
            isEdit={isEdit}
            onEdit={isPending ? () => handlePending('edit') : onEdit ? () => onEdit(true) : undefined}
            onClose={isEdit ? clickCancel : closeDrawer}
            onDelete={isPending ? () => handlePending(isSource ? 'uninstrument' : 'delete') : onEdit ? clickDelete : undefined}
            deleteLabel={isSource ? 'Uninstrument' : undefined}
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
