import { useEffect, useRef, useState } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import { LANGUAGES_LOGOS } from '@/utils';
import DrawerHeader from './drawer-header';
import DrawerFooter from './drawer-footer';
import { SourceDrawer } from '../../sources';
import { ActionDrawer } from '../../actions';
import { Drawer } from '@/reuseable-components';
import { DeleteEntityModal } from '@/components';
import getActionIcon from '@/utils/functions/get-action-icon';
import { useActualSources, useUpdateDestination } from '@/hooks';
import { DestinationDrawer, DestinationDrawerHandle } from '../../destinations';
import { getMainContainerLanguageLogo, WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils/constants/programming-languages';
import { WorkloadId, K8sActualSource, ActualDestination, OVERVIEW_ENTITY_TYPES, PatchSourceRequestInput, ActionDataParsed } from '@/types';

const componentMap = {
  source: SourceDrawer,
  action: ActionDrawer,
  destination: ({ isEditing }: { isEditing: boolean }) => <DestinationDrawer isEditing={isEditing} />,
};

const DRAWER_WIDTH = '640px';

const OverviewDrawer = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const setSelectedItem = useDrawerStore(({ setSelectedItem }) => setSelectedItem);

  const [isEditing, setIsEditing] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [title, setTitle] = useState('');

  const { updateExistingDestination } = useUpdateDestination();
  const { updateActualSource, deleteSourcesForNamespace } = useActualSources();

  const titleRef = useRef<HTMLInputElement>(null);
  const destinationDrawerRef = useRef<DestinationDrawerHandle>(null);

  useEffect(initialTitle, [selectedItem]);

  //TODO: split file to separate components by type: source, destination, action

  function initialTitle() {
    let str = '';

    if (!!selectedItem?.item) {
      const { type, item } = selectedItem;

      if (type === OVERVIEW_ENTITY_TYPES.SOURCE) {
        str = (item as K8sActualSource).reportedName;
      } else if (type === OVERVIEW_ENTITY_TYPES.ACTION) {
        str = (item as ActionDataParsed).spec.actionName;
      } else if (type === OVERVIEW_ENTITY_TYPES.DESTINATION) {
        str = (item as ActualDestination).name;
      }
    }

    setTitle(str);
  }

  const handleCancel = () => {
    setIsEditing(false);
    initialTitle();
  };

  const handleClose = () => {
    setIsEditing(false);
    setSelectedItem(null);
    setIsDeleteModalOpen(false);
  };

  const handleCloseDeleteModal = () => {
    setIsDeleteModalOpen(false);
  };

  const handleSave = async () => {
    if (!selectedItem?.item) return null;
    const { type, id, item } = selectedItem;

    if (type === OVERVIEW_ENTITY_TYPES.DESTINATION) {
      if (destinationDrawerRef.current && titleRef.current) {
        const newTitle = titleRef.current.value;
        const destinationData = {
          ...destinationDrawerRef.current.getCurrentData(),
          name: newTitle,
        };

        try {
          await updateExistingDestination(id as string, destinationData);
        } catch (error) {
          console.error('Error updating destination:', error);
        }
        setIsEditing(false);
      }
    }

    if (type === OVERVIEW_ENTITY_TYPES.ACTION) {
      alert('TODO !');
    }

    if (type === OVERVIEW_ENTITY_TYPES.SOURCE) {
      if (titleRef.current) {
        const newTitle = titleRef.current.value;
        setTitle(newTitle);

        const { namespace, name, kind } = item as K8sActualSource;

        const sourceId: WorkloadId = {
          namespace,
          kind,
          name,
        };

        const patchRequest: PatchSourceRequestInput = {
          reportedName: newTitle,
        };

        try {
          await updateActualSource(sourceId, patchRequest);
        } catch (error) {
          console.error('Error updating source:', error);
        }
      }
      setIsEditing(false);
    }
  };

  const handleDelete = async () => {
    if (!selectedItem?.item) return null;
    const { type, item } = selectedItem;

    if (type === OVERVIEW_ENTITY_TYPES.SOURCE) {
      const { namespace, name, kind } = item as K8sActualSource;

      try {
        await deleteSourcesForNamespace(namespace, [
          {
            kind,
            name,
            selected: false,
          },
        ]);
      } catch (error) {
        console.error('Error deleting source:', error);
      }
    }

    if (type === OVERVIEW_ENTITY_TYPES.ACTION) {
      alert('TODO !');
    }

    if (type === OVERVIEW_ENTITY_TYPES.DESTINATION) {
      alert('TODO !');
    }

    handleClose();
  };

  if (!selectedItem?.item) return null;

  const { type, item } = selectedItem;
  const SpecificComponent = componentMap[type];

  return SpecificComponent ? (
    <>
      <Drawer isOpen onClose={handleClose} width={DRAWER_WIDTH} closeOnEscape={!isDeleteModalOpen}>
        <DrawerContent>
          <DrawerHeader
            ref={titleRef}
            title={title}
            onClose={isEditing ? handleCancel : handleClose}
            imageUri={item ? getItemImageByType(type, item) : ''}
            isEditing={isEditing}
            setIsEditing={setIsEditing}
          />

          <ContentArea>
            {type === OVERVIEW_ENTITY_TYPES.DESTINATION ? (
              <DestinationDrawer ref={destinationDrawerRef} isEditing={isEditing} />
            ) : (
              <SpecificComponent isEditing={isEditing} />
            )}
          </ContentArea>

          {isEditing && <DrawerFooter onSave={handleSave} onCancel={handleCancel} onDelete={() => setIsDeleteModalOpen(true)} />}
        </DrawerContent>
      </Drawer>

      <DeleteEntityModal
        title={title}
        isModalOpen={isDeleteModalOpen}
        handleDelete={handleDelete}
        handleCloseModal={handleCloseDeleteModal}
        description='Are you sure you want to delete this source?'
      />
    </>
  ) : null;
};

function getItemImageByType(
  type: OVERVIEW_ENTITY_TYPES.SOURCE | OVERVIEW_ENTITY_TYPES.ACTION | OVERVIEW_ENTITY_TYPES.DESTINATION,
  item: K8sActualSource | ActionDataParsed | ActualDestination
): string {
  let src = '';

  switch (type) {
    case OVERVIEW_ENTITY_TYPES.SOURCE:
      src = getMainContainerLanguageLogo(item as K8sActualSource);
      break;

    case OVERVIEW_ENTITY_TYPES.ACTION:
      src = getActionIcon((item as ActionDataParsed).type);
      break;

    case OVERVIEW_ENTITY_TYPES.DESTINATION:
      src = (item as ActualDestination).destinationType.imageUrl;
      break;

    default:
      break;
  }

  return src || LANGUAGES_LOGOS[WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN];
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
