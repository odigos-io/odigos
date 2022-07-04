import { getConfiguration } from "@/utils/config";
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

function CollectorCard({ name, ready }: Collector) {
  return (
    <div className="shadow-lg border border-gray-200 rounded-lg bg-white hover:bg-gray-100 cursor-pointer">
      <Link href={`/collector/edit/${name}`}>
        <a className="flex flex-col items-start p-5">
          <div className="font-bold">{name}</div>
          {ready ? (
            <div className="text-green-600 font-medium">Ready</div>
          ) : (
            <div className="text-orange-400 font-medium">Not Ready</div>
          )}
        </a>
      </Link>
    </div>
  );
}

export const getServerSideProps = async () => {
  const config = await getConfiguration();
  if (!config) {
    return {
      redirect: {
        destination: "/setup",
        permanent: false,
      },
    };
  }

  return {
    props: {},
  };
};

export default CollectorsPage;
