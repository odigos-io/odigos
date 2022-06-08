import Sources from "@/components/Sources";
import Collectors from "@/components/Collectors";
import Destinations from "@/components/Destinations";

export default function Topology({}) {
  return (
    <div className="w-full h-full grid grid-cols-3">
      <div className="flex flex-col py-10 space-y-8">
        <div className="text-xl mx-auto">Sources</div>
        <div className="flex justify-center h-full">
          <Sources />
        </div>
      </div>
      <div className="flex flex-col">
        <div className="text-xl mx-auto pt-10">Collectors</div>
        <Collectors />
      </div>
      <div className="flex flex-col">
        <div className="text-xl mx-auto pt-10">Destinations</div>
        <Destinations />
      </div>
    </div>
  );
}
