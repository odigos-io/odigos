import { DestResponseItem } from "@/types/dests";
import Link from "next/link";
import Vendors from "@/vendors/index";

export default function EditDestCard({ name, type }: DestResponseItem) {
  const vendor = Vendors.find((v) => v.name === type);
  if (!vendor) {
    return null;
  }

  return (
    <div className="shadow-lg border border-gray-200 rounded-lg bg-white hover:bg-gray-100 cursor-pointer">
      <Link href={`/dest/${name}`}>
        <a className="flex flex-row p-3 items-center space-x-4">
          {vendor.getLogo({ className: "w-16 h-16" })}
          <div className="flex flex-col items-start">
            <div className="font-medium">{name}</div>
            <ul>
              {vendor.supportedSignals.map((signal) => {
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
