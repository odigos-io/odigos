import { useEffect, useState } from 'react';
import { useDrawerStore } from '@/store';
import { K8sActualSource } from '@/types';
import DrawerHeader from './drawer-header';
import DrawerFooter from './drawer-footer';
import { SourceDrawer } from '../../sources';
import { Drawer } from '@/reuseable-components';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';
import styled from 'styled-components';

const componentMap = {
  source: SourceDrawer,
  action: () => <div>Action</div>,
  destination: () => <div>Destination</div>,
};

const OverviewDrawer = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const setDrawerItem = useDrawerStore(
    ({ setSelectedItem }) => setSelectedItem
  );
  const [isEditing, setIsEditing] = useState(false);
  const [title, setTitle] = useState(selectedItem?.item?.name || '');

  useEffect(() => {
    console.log({ selectedItem });
    setTitle(selectedItem?.item?.name || '');
  }, [selectedItem]);

  const handleSaveTitle = (newTitle: string) => {
    setTitle(newTitle);
    // Add any save logic if needed
  };

  const handleSave = () => {
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

  if (!selectedItem) return null;

  const SpecificComponent = componentMap[selectedItem.type];

  return SpecificComponent ? (
    <Drawer isOpen onClose={() => setDrawerItem(null)} width="560px">
      <DrawerHeader
        title={title}
        imageUri={
          selectedItem?.item
            ? getMainContainerLanguageLogo(selectedItem.item as K8sActualSource) //TODO: get image based on type
            : ''
        }
        onSave={handleSaveTitle}
        {...{ isEditing, setIsEditing }}
      />
      <SpecificComponentWrapper>
        <SpecificComponent />
      </SpecificComponentWrapper>
      {isEditing && (
        <DrawerFooter
          onSave={handleSave}
          onCancel={handleCancel}
          onDelete={handleDelete}
        />
      )}
    </Drawer>
  ) : null;
};

export { OverviewDrawer };

const SpecificComponentWrapper = styled.div`
  padding: 20px 32px;
`;
