import { Menu } from "@headlessui/react";
import DatadogLogo from "@/img/vendor/datadog.svg";
import GrafanaLogo from "@/img/vendor/grafana.svg";
import HoneycombLogo from "@/img/vendor/honeycomb.svg";
import Link from "next/link";

export default function Destinations() {
  return (
    <div className="mx-auto mt-24">
      <AddDestinationCard />
    </div>
  );
}

function AddDestinationCard() {
  return (
    <Menu>
      <Menu.Button className="bg-white hover:bg-gray-100 shadow-lg border border-gray-200 rounded-lg w-64">
        <div className="flex flex-row items-center justify-center p-3 text-center">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            className="h-5 w-5"
            viewBox="0 0 20 20"
            fill="currentColor"
          >
            <path
              fillRule="evenodd"
              d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z"
              clipRule="evenodd"
            />
          </svg>
          <div>Add New Destination</div>
        </div>
      </Menu.Button>
      <Menu.Items className="flex flex-col mt-4 bg-white shadow-md border border-gray-200 divide-gray-200 divide-y">
        <Menu.Item>
          {({ active }) => (
            <Link href="/dest/new/grafana">
              <a
                className={`${
                  active && "bg-gray-100"
                } p-3 flex flex-row space-x-2 items-center`}
              >
                <GrafanaLogo className="w-10 h-10" />
                <div>Grafana</div>
              </a>
            </Link>
          )}
        </Menu.Item>
        <Menu.Item>
          {({ active }) => (
            <Link href="/dest/new/datadog">
              <a
                className={`${
                  active && "bg-gray-100"
                } p-3 flex flex-row space-x-2 items-center`}
              >
                <DatadogLogo className="w-10 h-10" />
                <div>Datadog</div>
              </a>
            </Link>
          )}
        </Menu.Item>
        <Menu.Item>
          {({ active }) => (
            <Link href="/dest/new/honeycomb">
              <a
                className={`${
                  active && "bg-gray-100"
                } p-3 flex flex-row space-x-2 items-center`}
              >
                <HoneycombLogo className="w-10 h-10" />
                <div>Honeycomb</div>
              </a>
            </Link>
          )}
        </Menu.Item>
      </Menu.Items>
    </Menu>
  );
}
