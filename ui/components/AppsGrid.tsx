import AppCard from "@/components/AppCard";
import type { ApplicationData, AppsApiResponse } from "@/types/apps";

interface AppsGridProps {
  apps: ApplicationData[];
  disabled: boolean;
  selectedApps: string[];
  setSelectedApps: any;
}

export default function AppsGrid({
  apps,
  disabled,
  selectedApps,
  setSelectedApps,
}: AppsGridProps) {
  const onSelection = (id: string, selection: boolean) => {
    const updatedSelectedApps =
      selection && !selectedApps.includes(id)
        ? [...selectedApps, id]
        : selectedApps.filter((e) => e !== id);
    setSelectedApps(updatedSelectedApps);
  };

  return (
    <div className="grid lg:grid-cols-3 2xl:grid-cols-4 max-w-7xl gap-4 pr-4">
      {apps &&
        apps.map((app: ApplicationData) => {
          return (
            <AppCard
              key={app.id}
              {...app}
              disabled={disabled}
              selected={selectedApps.includes(app.id)}
              onSelection={onSelection}
            />
          );
        })}
    </div>
  );
}
