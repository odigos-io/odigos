import { ObservabilityVendor } from "@/vendors/index";
import DestCard from "@/components/DestCard";

interface DestsGridProps {
  vendors: ObservabilityVendor[];
  title: string;
}

export default function DestsGrid({ vendors, title }: DestsGridProps) {
  return (
    <div className="space-y-3">
      <div className="text-2xl font-medium">{title}</div>
      <div className="grid lg:grid-cols-3 2xl:grid-cols-5 max-w-7xl gap-4 pr-4">
        {vendors.map((dest: ObservabilityVendor) => {
          return <DestCard key={dest.name} {...dest} />;
        })}
      </div>
    </div>
  );
}
