import { useEffect, useState } from 'react';
import { useDrawerStore } from '@/store';
import DrawerHeader from './drawer-header';
import DrawerFooter from './drawer-footer';
import { Drawer } from '@/reuseable-components';
import { SourceDrawer } from '../../sources';

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
        onSave={handleSaveTitle}
        {...{ isEditing, setIsEditing }}
      />
      <SpecificComponent />
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
