import { NextPage } from "next";
import type { KubernetesObjectsInNamespaces } from "@/types/apps";
import useSWR, { Fetcher } from "swr";
import { useState } from "react";
import { getConfiguration } from "@/utils/config";
import LoadingPage from "@/components/Loading";
import NamespaceSelector from "@/components/namespaces/Selector";

interface SetupProps {
  loading: boolean;
}

async function submitChanges(instMode: string, selectedApps: string[]) {
  const resp = await fetch("/api/appselector", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      instMode,
      selectedApps,
    }),
  });

  if (resp.ok) {
    window.location.href = "/dest/new";
  }
}

const SetupPage: NextPage = () => {
  const [instrumentationMode, setInstrumentationMode] = useState("OPT_OUT");
  const [selectedApps, setSelectedApps] = useState<string[]>([]);
  const fetcher: Fetcher<KubernetesObjectsInNamespaces, any> = (args: any) =>
    fetch(args).then((res) => res.json());
  const { data, error } = useSWR<KubernetesObjectsInNamespaces>("/api/v2/applications", fetcher);
  if (error) return <div>failed to load</div>;
  if (!data) return <LoadingPage />;

  return (
    <div>
      <div className="text-5xl mb-6">Choose target applications</div>
      <div>
        <div>Namespace:</div>
        <div>
          <NamespaceSelector />
        </div>

      </div>
      {/* <div className="pt-4">
        <AppsGrid
          apps={data.apps}
          disabled={instrumentationMode === "OPT_OUT"}
          selectedApps={selectedApps}
          setSelectedApps={setSelectedApps}
        />
      </div> */}
      <button
        type="button"
        onClick={() => submitChanges(instrumentationMode, selectedApps)}
        className="mt-6 text-white focus:ring-4 font-bold rounded-md text-sm px-14 py-2.5 mr-2 mb-2 bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-blue-800"
      >
        Save Changes
      </button>
    </div>
  );
};

export const getServerSideProps = async () => {
  const config = await getConfiguration();
  if (config) {
    return {
      redirect: {
        destination: "/",
        permanent: false,
      },
    };
  }

  return {
    props: {},
  };
};

export default SetupPage;
