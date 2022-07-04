import Vendors, { ObservabilityVendor } from "@/vendors/index";
import DestCard from "@/components/DestCard";

export default function DestsGrid() {
  return (
    <div className="grid lg:grid-cols-3 2xl:grid-cols-6 gap-4 pr-4">
      {Vendors.map((dest: ObservabilityVendor) => {
        return <DestCard key={dest.name} {...dest} />;
      })}
    </div>
  );
}
