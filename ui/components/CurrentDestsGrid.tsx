import { DestResponseItem } from "@/types/dests";
import useSWR, { Fetcher } from "swr";
import LoadingPage from "@/components/Loading";
import EditDestCard from "@/components/EditDestCard";

export default function CurrentDestsGrid() {
  const fetcher: Fetcher<DestResponseItem[], any> = (args: any) =>
    fetch(args).then((res) => res.json());
  const { data, error } = useSWR<DestResponseItem[]>("/api/dests", fetcher);
  if (error) return <div>failed to load</div>;
  if (!data) return <LoadingPage />;
  if (data.length == 0) {
    return (
      <div
        className="w-fit flex p-4 mb-4 text-sm text-blue-700 bg-blue-100 rounded-lg dark:bg-blue-200 dark:text-blue-800"
        role="alert"
      >
        <div>
          There are no configured destinations, use the top button to add new
          ones.
        </div>
      </div>
    );
  }

  return (
    <div className="grid lg:grid-cols-3 2xl:grid-cols-6 gap-4 pr-4">
      {data.map((dest) => {
        return <EditDestCard key={dest.name} {...dest} />;
      })}
    </div>
  );
}
