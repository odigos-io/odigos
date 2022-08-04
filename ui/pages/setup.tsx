import Alert from "@/components/Alert";
import AppsGrid from "@/components/AppsGrid";
import { NextPage } from "next";
import type { AppsApiResponse } from "@/types/apps";
import useSWR, { Fetcher } from "swr";
import { useState } from "react";
import { getConfiguration } from "@/utils/config";
import LoadingPage from "@/components/Loading";

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
  const fetcher: Fetcher<AppsApiResponse, any> = (args: any) =>
    fetch(args).then((res) => res.json());
  const { data, error } = useSWR<AppsApiResponse>("/api/apps", fetcher, {
    refreshInterval: 2000,
  });
  if (error) return <div>failed to load</div>;
  if (!data) return <LoadingPage />;

  return (
    <div>
      <div className="text-5xl mb-6">Choose target applications</div>
      <div>
        Please select how odigos should choose which applications to instrument
        <div className="flex items-center mt-4 ml-2">
          <input
            checked={instrumentationMode === "OPT_OUT"}
            id="OPT_OUT"
            type="radio"
            value="OPT_OUT"
            onChange={(e) => {
              setInstrumentationMode(e.currentTarget.value);
              setSelectedApps([]);
            }}
            name="instrumentation-mode"
            className="w-5 h-5 text-blue-600 bg-gray-100 border-gray-300 focus:ring-blue-500 focus:ring-2"
          />
          <label htmlFor="OPT_OUT" className="ml-2 text-md">
            Instrument any applications found, new application will be
            instrumented automatically (Opt out)
          </label>
        </div>
        <div className="flex items-center mt-1 ml-2">
          <input
            id="OPT_IN"
            checked={instrumentationMode === "OPT_IN"}
            type="radio"
            value="OPT_IN"
            onChange={(e) => setInstrumentationMode(e.currentTarget.value)}
            name="instrumentation-mode"
            className="w-5 h-5 text-blue-600 bg-gray-100 border-gray-300 focus:ring-blue-500 focus:ring-2"
          />
          <label htmlFor="OPT_IN" className="ml-2 text-md">
            Instrument only the selected applications, new applications will not
            be instrumented automatically (Opt in)
          </label>
        </div>
      </div>
      {data.discovery_in_progress && (
        <div className="pt-4">
          <Alert message="Applications discovery in progress, this should take a few seconds..." />
        </div>
      )}
      <div className="pt-4">
        <AppsGrid
          apps={data.apps}
          disabled={instrumentationMode === "OPT_OUT"}
          selectedApps={selectedApps}
          setSelectedApps={setSelectedApps}
        />
      </div>
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
