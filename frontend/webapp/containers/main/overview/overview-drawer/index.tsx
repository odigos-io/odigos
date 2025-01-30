import { PropsWithChildren, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react';
import { useTheme } from 'styled-components';
import { useDestinationCRUD, useSourceCRUD } from '@/hooks';
import { CancelWarning, DeleteWarning } from '@/components/modals';
import { NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES } from '@/types';
import { useDrawerStore, useNotificationStore, usePendingStore } from '@/store';
import { Drawer, DrawerProps, EditIcon, Input, Text, TrashIcon, Types, useKeyDown } from '@odigos/ui-components';

interface OverviewDrawerProps {
  title: string;
  titleTooltip?: string;
  icon?: Types.SVG;
  iconSrc?: string;
  isEdit?: boolean;
  isFormDirty?: boolean;
  onEdit?: (bool?: boolean) => void;
  onSave?: (newTitle: string) => void;
  onDelete?: () => void;
  onCancel?: () => void;
}

export interface DrawerHeaderRef {
  getTitle: () => string;
  clearTitle: () => void;
}

const DRAWER_WIDTH = `${640 + 64}px`; // +64 because of "ContentArea" padding

const OverviewDrawer: React.FC<OverviewDrawerProps & PropsWithChildren> = ({
  children,
  title,
  titleTooltip,
  icon,
  iconSrc,
  isEdit = false,
  isFormDirty = false,
  onEdit,
  onSave,
  onDelete,
  onCancel,
}) => {
  const theme = useTheme();
  const { isThisPending } = usePendingStore();
  const { addNotification } = useNotificationStore();
  const { selectedItem, setSelectedItem } = useDrawerStore();

  useKeyDown({ key: 'Enter', active: !!selectedItem?.item }, () => (isEdit ? clickSave() : closeDrawer()));

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

  const [inputValue, setInputValue] = useState(title);

  useEffect(() => {
    setInputValue(title);
  }, [title]);

  useImperativeHandle(titleRef, () => ({
    getTitle: () => inputValue,
    clearTitle: () => setInputValue(title),
  }));

  const actionButtons: DrawerProps['header']['actionButtons'] = [];

  if (!!onEdit && !isEdit)
    actionButtons.push({
      'data-id': 'drawer-edit',
      variant: 'tertiary',
      onClick: isPending ? () => handlePending('edit') : onEdit ? () => onEdit(true) : undefined,
      children: (
        <>
          <EditIcon />
          <Text size={14} family='secondary' decoration='underline'>
            Edit
          </Text>
        </>
      ),
    });

  if (!!onDelete && !isEdit)
    actionButtons.push({
      'data-id': 'drawer-delete',
      variant: 'tertiary',
      onClick: isPending ? () => handlePending(isSource ? 'uninstrument' : 'delete') : onEdit ? clickDelete : undefined,
      children: (
        <>
          <TrashIcon />
          <Text color={theme.text.error} size={14} family='secondary' decoration='underline'>
            {isSource ? 'Uninstrument' : 'Delete'}
          </Text>
        </>
      ),
    });

  return (
    <>
      <Drawer
        isOpen
        onClose={isEdit ? clickCancel : closeDrawer}
        closeOnEscape={!isDeleteModalOpen && !isCancelModalOpen}
        width={DRAWER_WIDTH}
        header={{
          icon,
          iconSrc,
          title,
          titleTooltip,
          replaceTitleWith: !isSource && isEdit ? () => <Input data-id='title' autoFocus value={inputValue} onChange={(e) => setInputValue(e.target.value)} /> : undefined,
          actionButtons,
        }}
        footer={{
          isOpen: isEdit,
          leftButtons: [
            {
              'data-id': 'drawer-save',
              variant: 'primary',
              onClick: clickSave,
              children: 'save',
            },
            {
              'data-id': 'drawer-cancel',
              variant: 'secondary',
              onClick: clickCancel,
              children: 'cancel',
            },
          ],
          rightButtons: [
            {
              'data-id': 'drawer-delete',
              variant: 'tertiary',
              onClick: clickDelete,
              children: (
                <>
                  <TrashIcon />
                  <Text size={14} color={theme.text.error} family='secondary' decoration='underline'>
                    delete
                  </Text>
                </>
              ),
            },
          ],
        }}
      >
        {children}
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
