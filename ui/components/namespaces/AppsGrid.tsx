import AppCard from "@/components/namespaces/AppCard";
import type { KubernetesObject, KubernetesNamespace } from "@/types/apps";

interface AppsGridProps {
  selectedNamespace: KubernetesNamespace;
  changeObjectLabel: (obj: KubernetesObject) => void;
}

export default function AppsGrid({
  selectedNamespace,
  changeObjectLabel
}: AppsGridProps) {
  if (selectedNamespace && selectedNamespace.objects.length === 0) {
    return (
      <div className="text-center shadow-lg border border-gray-200 rounded-lg bg-gray-100 p-5 font-light text-gray-900 w-fit">
        This namespace has no applications</div>
    );
  }

  return (
    <div className="grid lg:grid-cols-3 2xl:grid-cols-4 max-w-7xl gap-4 pr-4">
      {selectedNamespace &&
        selectedNamespace.objects.map((app: KubernetesObject) => {
          return (
            <AppCard
              key={`${app.kind}-${app.name}`}
              disabled={selectedNamespace.labeled}
              selected={app.labeled}
              object={app}
              onSelection={changeObjectLabel}
            />
          );
        })}
    </div>
  );
}
