import EditAppsGrid from "@/components/EditAppsGrid";
import LoadingPage from "@/components/Loading";
import { AppsApiResponse } from "@/types/apps";
import type { NextPage } from "next";
import useSWR, { Fetcher } from "swr";

const SourcesPage: NextPage = () => {
  const fetcher: Fetcher<AppsApiResponse, any> = (args: any) =>
    fetch(args).then((res) => res.json());
  const { data, error } = useSWR<AppsApiResponse>("/api/apps", fetcher);
  if (error) return <div>failed to load</div>;
  if (!data) return <LoadingPage />;

  return (
    <div className="space-y-12">
      <div className="text-4xl font-medium">Active Sources</div>
      <EditAppsGrid {...data} />
    </div>
  );
};

export default SourcesPage;
