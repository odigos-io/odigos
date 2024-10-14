import { useEffect, useRef, useState } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import DrawerHeader from './drawer-header';
import DrawerFooter from './drawer-footer';
import { SourceDrawer } from '../../sources';
import { Drawer } from '@/reuseable-components';
import { DeleteEntityModal } from '@/components';
import { useActualSources, useUpdateDestination } from '@/hooks';
import { DestinationDrawer, DestinationDrawerHandle } from '../../destinations';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';
import {
  WorkloadId,
  K8sActualSource,
  ActualDestination,
  PatchSourceRequestInput,
} from '@/types';

const componentMap = {
  source: SourceDrawer,
  action: () => <div>Action</div>,
  destination: (props: { isEditing: boolean }) => (
    <DestinationDrawer isEditing={props.isEditing} />
  ),
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

  const { updateExistingDestination } = useUpdateDestination();
  const { updateActualSource, deleteSourcesForNamespace } = useActualSources();
  const titleRef = useRef<HTMLInputElement>(null);
  const destinationDrawerRef = useRef<DestinationDrawerHandle>(null);
  useEffect(initialTitle, [selectedItem]);

  function initialTitle() {
    if (selectedItem?.type === 'source' && selectedItem.item) {
      const title = (selectedItem.item as K8sActualSource).reportedName;
      setTitle(title || '');
    } else if (selectedItem?.type === 'destination' && selectedItem.item) {
      const title = (selectedItem.item as ActualDestination).name;
      setTitle(title || '');
    } else {
      setTitle('');
    }
  }

  const handleSave = async () => {
    if (selectedItem?.type === 'destination') {
      if (destinationDrawerRef.current && titleRef.current) {
        const name = titleRef.current.value;
        const destinationData = {
          ...destinationDrawerRef.current.getCurrentData(),
          name,
        };
        try {
          const res = await updateExistingDestination(
            selectedItem.id as string,
            destinationData
          );
          console.log({ res });
        } catch (error) {
          console.error('Error updating destination:', error);
        }

        try {
          // Replace this with your actual save logic
          // await updateDestination(destinationData);
        } catch (error) {
          console.error('Error updating destination:', error);
          // Optionally show error message to user
        }
        setIsEditing(false);
      }
    }

    if (selectedItem?.type === 'source') {
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
    }
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
    <>
      <Drawer
        isOpen
        onClose={handleClose}
        width={DRAWER_WIDTH}
        closeOnEscape={!isDeleteModalOpen}
      >
        <DrawerContent>
          <DrawerHeader
            ref={titleRef}
            title={title}
            onClose={handleClose}
            imageUri={
              selectedItem?.item ? getItemImageByType(selectedItem?.item) : ''
            }
            {...{ isEditing, setIsEditing }}
          />
          <ContentArea>
            {selectedItem.type === 'destination' ? (
              <DestinationDrawer
                ref={destinationDrawerRef}
                isEditing={isEditing}
              />
            ) : (
              <SpecificComponent isEditing={isEditing} />
            )}
          </ContentArea>
          {isEditing && (
            <>
              <DrawerFooter
                onSave={handleSave}
                onCancel={handleCancel}
                onDelete={() => setIsDeleteModalOpen(true)}
              />
            </>
          )}
        </DrawerContent>
      </Drawer>
      <DeleteEntityModal
        title={title}
        isModalOpen={isDeleteModalOpen}
        handleDelete={handleDelete}
        handleCloseModal={handleCloseDeleteModal}
        description="Are you sure you want to delete this source?"
      />
    </>
  ) : null;
};

function getItemImageByType(item: K8sActualSource | ActualDestination): string {
  if ('destinationType' in item) {
    // item is of type ActualDestination
    return item.destinationType.imageUrl;
  } else {
    // item is of type K8sActualSource
    return getMainContainerLanguageLogo(item as K8sActualSource);
  }
}

export default OverviewDrawer;

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
