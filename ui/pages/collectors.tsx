import type { NextPage } from "next";
import type { Collector, ICollectorsResponse } from "@/types/collectors";
import useSWR, { Key, Fetcher } from "swr";
import LoadingPage from "@/components/Loading";
import Link from "next/link";

const CollectorsPage: NextPage = () => {
  const fetcher: Fetcher<ICollectorsResponse, any> = (args: any) =>
    fetch(args).then((res) => res.json());
  const { data, error } = useSWR<ICollectorsResponse>(
    "/api/collectors",
    fetcher
  );
  if (error) return <div>failed to load</div>;
  if (!data) return <LoadingPage />;

  return (
    <div className="space-y-12">
      <div className="text-4xl font-medium">Active Collectors</div>
      {data ? (
        <div className="grid lg:grid-cols-3 2xl:grid-cols-6 gap-4 pr-4">
          {data.collectors.map((collector) => {
            return <CollectorCard key={collector.name} {...collector} />;
          })}
        </div>
      ) : (
        <NoCollectorsCard />
      )}
    </div>
  );
};

function NoCollectorsCard() {
  return (
    <div className="mx-auto cursor-not-allowed mt-24 bg-gray-100 shadow-lg border border-gray-200 rounded-lg w-64">
      <div className="flex flex-col items-center justify-center p-3 text-center">
        <div>Collectors not deployed yet.</div>
        <div>Configure destinations to start</div>
      </div>
    </div>
  );
}

async function deleteCollector(name: string) {
  const response = await fetch(`/api/collectors`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    
    body: JSON.stringify({ name }),
  });

  if (!response.ok) {
    throw new Error(`Failed to delete collector ${name}`);
  }
}

function CollectorCard({ name, ready }: Collector) {
  return (
    <div className="shadow-lg border border-gray-200 rounded-lg bg-white">
      <div className="flex flex-col items-start p-5">
        <div className="font-bold">{name}</div>
        <div className="flex flex-row justify-between w-full">
          {ready ? (
            <div className="text-green-600 font-medium">Ready</div>
          ) : (
            <div className="text-orange-400 font-medium">Not Ready</div>
          )}
          <button
            onClick={() => deleteCollector(name)}
            className="hover:bg-gray-100 cursor-pointer p-1 rounded-lg"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-5 w-5"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path
                fillRule="evenodd"
                d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}


export default CollectorsPage;
