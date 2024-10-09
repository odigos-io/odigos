import { useEffect, useRef, useState } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import { useActualSources } from '@/hooks';
import DrawerHeader from './drawer-header';
import DrawerFooter from './drawer-footer';
import { SourceDrawer } from '../../sources';
import { Drawer } from '@/reuseable-components';
import { DeleteEntityModal } from '@/components';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';
import { K8sActualSource, PatchSourceRequestInput, WorkloadId } from '@/types';

const componentMap = {
  source: SourceDrawer,
  action: () => <div>Action</div>,
  destination: () => <div>Destination</div>,
};

const DRAWER_WIDTH = '560px';

const OverviewDrawer = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const setDrawerItem = useDrawerStore(
    ({ setSelectedItem }) => setSelectedItem
  );
  const [isEditing, setIsEditing] = useState(false);
  const [title, setTitle] = useState(selectedItem?.item?.name || '');
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);

  const { updateActualSource, deleteSourcesForNamespace } = useActualSources();

  const titleRef = useRef<HTMLInputElement>(null);

  useEffect(initialTitle, [selectedItem]);

  function initialTitle() {
    if (selectedItem?.type === 'source' && selectedItem.item) {
      const title = (selectedItem.item as K8sActualSource).reportedName;
      setTitle(title || '');
    } else {
      setTitle('');
    }
  }

  const handleSave = async () => {
    if (titleRef.current) {
      const newTitle = titleRef.current.value;
      setTitle(newTitle);
      if (selectedItem?.type === 'source' && selectedItem.item) {
        const sourceItem = selectedItem.item as K8sActualSource;

        const sourceId: WorkloadId = {
          namespace: sourceItem.namespace,
          kind: sourceItem.kind,
          name: sourceItem.name,
        };

        const patchRequest: PatchSourceRequestInput = {
          reportedName: newTitle,
        };

        try {
          await updateActualSource(sourceId, patchRequest);
        } catch (error) {
          console.error('Error updating source:', error);
          // Optionally show error message to user
        }
      }
    }
    setIsEditing(false);
  };

  const handleCancel = () => {
    setIsEditing(false);
    initialTitle();
  };

  const handleDelete = async () => {
    if (selectedItem?.type === 'source' && selectedItem.item) {
      const sourceItem = selectedItem.item as K8sActualSource;

      try {
        await deleteSourcesForNamespace(sourceItem.namespace, [
          {
            kind: sourceItem.kind,
            name: sourceItem.name,
            selected: false,
          },
        ]);
      } catch (error) {
        console.error('Error deleting source:', error);
      }
    }
    setDrawerItem(null); // Close the drawer on delete
  };

  const handleClose = () => {
    setIsEditing(false);
    setDrawerItem(null);
    setIsDeleteModalOpen(false);
  };

  const handleCloseDeleteModal = () => {
    setIsDeleteModalOpen(false);
  };

  if (!selectedItem) return null;

  const SpecificComponent = componentMap[selectedItem.type];

  return SpecificComponent ? (
    <Drawer isOpen onClose={handleClose} width={DRAWER_WIDTH}>
      <DrawerContent>
        <DrawerHeader
          ref={titleRef}
          title={title}
          imageUri={
            selectedItem?.item
              ? getMainContainerLanguageLogo(
                  selectedItem.item as K8sActualSource
                )
              : ''
          }
          {...{ isEditing, setIsEditing }}
        />
        <ContentArea>
          <SpecificComponent />
        </ContentArea>
        {isEditing && (
          <>
            <DrawerFooter
              onSave={handleSave}
              onCancel={handleCancel}
              onDelete={() => setIsDeleteModalOpen(true)}
            />
            <DeleteEntityModal
              title={title}
              isModalOpen={isDeleteModalOpen}
              handleDelete={handleDelete}
              handleCloseModal={handleCloseDeleteModal}
              description="Are you sure you want to delete this source?"
            />
          </>
        )}
      </DrawerContent>
    </Drawer>
  ) : null;
};

export { OverviewDrawer };

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
