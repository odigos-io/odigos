import { useEffect, useRef, useState } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import { K8sActualSource } from '@/types';
import DrawerHeader from './drawer-header';
import DrawerFooter from './drawer-footer';
import { SourceDrawer } from '../../sources';
import { Drawer } from '@/reuseable-components';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';

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

  const titleRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setTitle(selectedItem?.item?.name || '');
  }, [selectedItem]);

  const handleSave = () => {
    if (titleRef.current) {
      const newTitle = titleRef.current.value;
      setTitle(newTitle);
    }
    // Add save logic here
    setIsEditing(false);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setTitle(selectedItem?.item?.name || ''); // Revert to original title on cancel
  };

  const handleDelete = () => {
    // Add delete logic here
    setDrawerItem(null); // Close the drawer on delete
  };

  const handleClose = () => {
    setIsEditing(false);
    setDrawerItem(null);
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
          <DrawerFooter
            onSave={handleSave}
            onCancel={handleCancel}
            onDelete={handleDelete}
          />
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
