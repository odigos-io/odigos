import { useDrawerStore } from '@/store';

const componentMap = {
  source: () => <div>Source</div>,
  action: () => <div>Action</div>,
  destination: () => <div>Destination</div>,
};

const OverviewDrawer = () => {
  const selectedItem = useDrawerStore((state) => state.selectedItem);

  if (!selectedItem) return null;

  const SpecificComponent = componentMap[selectedItem.type];

  return SpecificComponent ? (
    <SpecificComponent />
  ) : (
    <div>Component not found</div>
  );
};

export { OverviewDrawer };
