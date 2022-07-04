import type { ApplicationData, AppsApiResponse } from "@/types/apps";
import EditAppCard from "@/components/EditAppCard";

export default function EditAppsGrid({ apps }: AppsApiResponse) {
  return (
    <div className="grid lg:grid-cols-3 2xl:grid-cols-6 gap-4 pr-4">
      {apps &&
        apps.map((app: ApplicationData) => {
          return <EditAppCard key={app.id} {...app} />;
        })}
    </div>
  );
}
