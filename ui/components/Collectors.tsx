import type { ICollectorsResponse } from "@/types/collectors";
import useSWR, { Key, Fetcher } from "swr";

export default function Collectors() {
  const fetcher: Fetcher<ICollectorsResponse, any> = (args: any) =>
    fetch(args).then((res) => res.json());
  const { data, error } = useSWR<ICollectorsResponse>(
    "/api/collectors",
    fetcher
  );
  if (error) return <div>failed to load</div>;
  if (!data) return <div>loading...</div>;

  if (data.total === 0) {
    return <NoCollectorsCard />;
  }

  return (
    <div>
      <div className="mx-auto mt-24 shadow-lg border border-gray-200 rounded-lg w-64">
        <div className="flex flex-col items-center justify-center p-3 text-center">
          <div>
            {data.ready}/{data.total} Collectors running
          </div>
        </div>
      </div>
    </div>
  );
}


