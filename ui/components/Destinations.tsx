import { Menu } from "@headlessui/react";
import DatadogLogo from "@/img/vendor/datadog.svg";
import GrafanaLogo from "@/img/vendor/grafana.svg";
import HoneycombLogo from "@/img/vendor/honeycomb.svg";
import Link from "next/link";
import { forwardRef } from "react";
import type { DestResponseItem } from "@/types/dests";
import useSWR, { Key, Fetcher } from "swr";

export default function Destinations() {
  const fetcher: Fetcher<DestResponseItem[], any> = (args: any) =>
    fetch(args).then((res) => res.json());
  const { data, error } = useSWR<DestResponseItem[]>("/api/dests", fetcher);
  if (error) return <div>failed to load</div>;
  if (!data) return <div>loading...</div>;

  return (
    <div className="mx-auto mt-24">
      <div className="flex flex-col space-y-8">
        {data.length > 0 &&
          data.map((dest) => {
            return (
              <DestinationCard
                key={dest.id}
                name={dest.name}
                type={dest.type}
              />
            );
          })}
        <AddDestinationCard />
      </div>
    </div>
  );
}

function DestinationCard({ name, type }: { name: string; type: string }) {
  return (
    <div className="bg-white hover:bg-gray-100 shadow-lg border border-gray-200 rounded-lg w-64">
      <div className="flex flex-col items-center justify-center p-3 text-center">
        <div>Name: {name}</div>
        <div>Type: {type}</div>
      </div>
    </div>
  );
}

const MenuLink = forwardRef((props: any, ref) => {
  let { href, children, ...rest } = props;
  return (
    <Link href={href}>
      <a ref={ref} {...rest}>
        {children}
      </a>
    </Link>
  );
});

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
            <MenuLink
              href="/dest/new/grafana"
              className={`${
                active && "bg-gray-100"
              } p-3 flex flex-row space-x-2 items-center`}
            >
              <GrafanaLogo className="w-10 h-10" />
              <div>Grafana</div>
            </MenuLink>
          )}
        </Menu.Item>
        <Menu.Item>
          {({ active }) => (
            <MenuLink
              href="/dest/new/datadog"
              className={`${
                active && "bg-gray-100"
              } p-3 flex flex-row space-x-2 items-center`}
            >
              <DatadogLogo className="w-10 h-10" />
              <div>Datadog</div>
            </MenuLink>
          )}
        </Menu.Item>
        <Menu.Item>
          {({ active }) => (
            <MenuLink
              href="/dest/new/honeycomb"
              className={`${
                active && "bg-gray-100"
              } p-3 flex flex-row space-x-2 items-center`}
            >
              <HoneycombLogo className="w-10 h-10" />
              <div>Honeycomb</div>
            </MenuLink>
          )}
        </Menu.Item>
      </Menu.Items>
    </Menu>
  );
}
