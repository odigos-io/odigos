import { useDrawerStore } from '@/store';
import { Drawer } from '@/reuseable-components';

const componentMap = {
  source: () => <div>Source</div>,
  action: () => <div>Action</div>,
  destination: () => <div>Destination</div>,
};

const OverviewDrawer = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const setDrawerItem = useDrawerStore(
    ({ setSelectedItem }) => setSelectedItem
  );

  if (!selectedItem) return null;

  const SpecificComponent = componentMap[selectedItem.type];

  return SpecificComponent ? (
    <Drawer isOpen onClose={() => setDrawerItem(null)}>
      <SpecificComponent />
    </Drawer>
  ) : null;
};

export { OverviewDrawer };
