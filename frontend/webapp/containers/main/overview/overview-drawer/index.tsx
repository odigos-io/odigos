import { forwardRef, type PropsWithChildren, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react';
import Theme from '@odigos/ui-theme';
import { useDestinationCRUD, useSourceCRUD } from '@/hooks';
import { EditIcon, TrashIcon, type SVG } from '@odigos/ui-icons';
import { ENTITY_TYPES, NOTIFICATION_TYPE, useKeyDown, WorkloadId } from '@odigos/ui-utils';
import { useDrawerStore, useNotificationStore, usePendingStore } from '@odigos/ui-containers';
import { CancelWarning, DeleteWarning, Drawer, DrawerProps, Input, Text } from '@odigos/ui-components';

interface OverviewDrawerProps {
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

interface EditTitleRef {
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
  const theme = Theme.useTheme();
  const { isThisPending } = usePendingStore();
  const { addNotification } = useNotificationStore();
  const { drawerType, drawerEntityId, setDrawerType, setDrawerEntityId } = useDrawerStore();

  useKeyDown({ key: 'Enter', active: isEdit }, () => clickSave());

  const { sources } = useSourceCRUD();
  const { destinations } = useDestinationCRUD();

  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isCancelModalOpen, setIsCancelModalOpen] = useState(false);

  const titleRef = useRef<EditTitleRef>(null);

  const isSource = drawerType === ENTITY_TYPES.SOURCE;
  const isDestination = drawerType === ENTITY_TYPES.DESTINATION;

  const closeDrawer = () => {
    setDrawerType(null);
    setDrawerEntityId(null);
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
    if (!drawerType) return false;

    return isThisPending({
      entityType: drawerType as ENTITY_TYPES,
      entityId: drawerEntityId as string | WorkloadId,
    });
  }, [drawerType, drawerEntityId]);

  const handlePending = (action: string) => {
    addNotification({
      type: NOTIFICATION_TYPE.WARNING,
      title: 'Pending',
      message: `Cannot click ${action}, ${drawerType} is pending`,
      hideFromHistory: true,
    });
  };

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
          replaceTitleWith: !isSource && isEdit ? () => <EditTitle ref={titleRef} title={title} /> : undefined,
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
        name={`${drawerType}${title ? ` (${title})` : ''}`}
        type={drawerType as ENTITY_TYPES}
        isLastItem={isLastItem()}
        onApprove={handleDelete}
        onDeny={closeWarningModals}
      />
      <CancelWarning isOpen={isCancelModalOpen} noOverlay name='edit mode' onApprove={handleCancel} onDeny={closeWarningModals} />
    </>
  );
};

const EditTitle = forwardRef<EditTitleRef, { title: string }>(({ title }, ref) => {
  const [inputValue, setInputValue] = useState(title);

  useEffect(() => {
    setInputValue(title);
  }, [title]);

  useImperativeHandle(ref, () => ({
    getTitle: () => inputValue,
    clearTitle: () => setInputValue(title),
  }));

  return <Input data-id='title' autoFocus value={inputValue} onChange={(e) => setInputValue(e.target.value)} />;
});

export default OverviewDrawer;
