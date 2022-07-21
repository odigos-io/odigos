import { DestResponseItem } from "@/types/dests";
import { ObservabilityVendor } from "@/vendors/index";
import Link from "next/link";

export default function DestCard({
  name,
  displayName,
  getLogo,
  supportedSignals,
}: ObservabilityVendor) {
  return (
    <div className="shadow-lg border border-gray-200 rounded-lg bg-white hover:bg-gray-100 cursor-pointer">
      <Link href={`/dest/new/${name}`}>
        <a className="flex flex-row p-3 items-top space-x-4">
          {getLogo({ className: "w-16 h-16" })}
          <div className="flex flex-col items-start">
            <div className="font-medium">{displayName}</div>
            <ul>
              {supportedSignals.map((signal) => {
                return (
                  <li key={signal} className="text-sm">
                    {signal}
                  </li>
                );
              })}
            </ul>
          </div>
        </a>
      </Link>
    </div>
  );
}
