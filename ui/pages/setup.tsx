import Alert from "@/components/Alert";
import AppsGrid from "@/components/AppsGrid";
import Sidebar from "@/components/Sidebar";
import { NextPage } from "next";
import type { AppsApiResponse } from "@/types/apps";
import useSWR, { Key, Fetcher } from "swr";
import Spinner from "@/components/Spinner";
import { useState } from "react";

interface SetupProps {
  loading: boolean;
}

function LoadingPage() {
  return (
    <div className="flex items-center justify-center w-screen h-screen">
      <Spinner className="w-12 h-12" />
    </div>
  );
}

function submitChanges(instMode: string, selectedApps: string[]) {
  console.log(`Inst mode is ${instMode} selected apps: ${selectedApps}`);
}

const SetupPage: NextPage = () => {
  const [instrumentationMode, setInstrumentationMode] = useState("opt-out");
  const [selectedApps, setSelectedApps] = useState<string[]>([]);
  const fetcher: Fetcher<AppsApiResponse, any> = (args: any) =>
    fetch(args).then((res) => res.json());
  const { data, error } = useSWR<AppsApiResponse>("/api/apps", fetcher, {
    refreshInterval: 2000,
  });
  if (error) return <div>failed to load</div>;
  if (!data) return <LoadingPage />;

  return (
    <div className="flex flex-row antialiased bg-white">
      <Sidebar />
      <div className="pt-10 pl-5 w-full text-gray-700 text-xl">
        <div className="text-5xl mb-6">Choose target applications</div>
        <div>
          Please select how odigos should choose which applications to
          instrument
          <div className="flex items-center mt-4 ml-2">
            <input
              checked={
                !data.discovery_in_progress && instrumentationMode === "opt-out"
              }
              id="opt-out"
              disabled={data.discovery_in_progress}
              type="radio"
              value="opt-out"
              onChange={(e) => {
                setInstrumentationMode(e.currentTarget.value);
                setSelectedApps([]);
              }}
              name="instrumentation-mode"
              className="w-5 h-5 text-blue-600 bg-gray-100 border-gray-300 focus:ring-blue-500 focus:ring-2"
            />
            <label
              htmlFor="opt-out"
              className={
                "ml-2 text-md " +
                (data.discovery_in_progress ? "text-gray-500" : "")
              }
            >
              Instrument any applications found, new application will be
              instrumented automatically (Opt out)
            </label>
          </div>
          <div className="flex items-center mt-1 ml-2">
            <input
              id="opt-in"
              disabled={data.discovery_in_progress}
              checked={
                !data.discovery_in_progress && instrumentationMode === "opt-in"
              }
              type="radio"
              value="opt-in"
              onChange={(e) => setInstrumentationMode(e.currentTarget.value)}
              name="instrumentation-mode"
              className="w-5 h-5 text-blue-600 bg-gray-100 border-gray-300 focus:ring-blue-500 focus:ring-2"
            />
            <label
              htmlFor="opt-in"
              className={
                "ml-2 text-md " +
                (data.discovery_in_progress ? "text-gray-500" : "")
              }
            >
              Instrument only the selected applications, new applications will
              not be instrumented automatically (Opt in)
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
            disabled={
              data.discovery_in_progress || instrumentationMode === "opt-out"
            }
            selectedApps={selectedApps}
            setSelectedApps={setSelectedApps}
          />
        </div>
        {!data.discovery_in_progress && (
          <button
            type="button"
            onClick={() => submitChanges(instrumentationMode, selectedApps)}
            className="mt-6 text-white focus:ring-4 font-bold rounded-md text-sm px-14 py-2.5 mr-2 mb-2 bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-blue-800"
          >
            Save Changes
          </button>
        )}
      </div>
    </div>
  );
};

export default SetupPage;
