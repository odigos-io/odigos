import { useDrawerStore } from '@/store';
import { Drawer } from '@/reuseable-components';
import { SourceDrawer } from '@/containers';
import DrawerHeader from './drawer-header';
import DrawerFooter from './drawer-footer';
import { useEffect, useState } from 'react';

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
    <Drawer isOpen onClose={() => setDrawerItem(null)}>
      <DrawerHeader title={title} onSave={handleSaveTitle} />
      <SpecificComponent />
      <DrawerFooter
        onSave={handleSave}
        onCancel={handleCancel}
        onDelete={handleDelete}
      />
      {/* {isEditing && (
      )} */}
    </Drawer>
  ) : null;
};

export { OverviewDrawer };
