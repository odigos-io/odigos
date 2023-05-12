import { NextPage } from "next";
import type { KubernetesNamespace, KubernetesObjectsInNamespaces } from "@/types/apps";
import useSWR, { Fetcher } from "swr";
import { useState } from "react";
import LoadingPage from "@/components/Loading";
import NamespaceSelector from "@/components/namespaces/Selector";
import AppsGrid from "@/components/namespaces/AppsGrid";
import { Switch } from '@headlessui/react'

const emptyNamespace: KubernetesNamespace = {
  name: "namespaces not found",
  labeled: false,
  objects: [],
}

async function submitChanges(data: KubernetesObjectsInNamespaces | undefined) {
  if (!data) return;

  const resp = await fetch("/api/appselector", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      data
    }),
  });

  if (resp.ok) {
    window.location.href = "/dest/new";
  }
}

const SetupPage: NextPage = () => {
  const [instrumentationMode, setInstrumentationMode] = useState("OPT_OUT");
  const [selectedApps, setSelectedApps] = useState<string[]>([]);
  const [appSelection, setAppSelection] = useState<KubernetesObjectsInNamespaces>();
  const fetcher: Fetcher<KubernetesObjectsInNamespaces, any> = (args: any) =>
    fetch(args).then((res) => res.json());
  const { data, error } = useSWR<KubernetesObjectsInNamespaces>("/api/v2/applications", fetcher);
  const [selectedNamespace, setSelectedNamespace] = useState(emptyNamespace)
  if (error) return <div>failed to load</div>;
  if (!data) return <LoadingPage />;
  if (!appSelection && data) {
    setAppSelection(data);
    setSelectedNamespace(data.namespaces[0]);
  }
  return (
    <div>
      <div className="text-5xl mb-6">Choose Target Applications</div>
      <div className="flex flex-col space-y-2">
        <div className="text-xl font-light mb-2">Select namespace:</div>
        <div className="flex flex-row space-x-10 items-baseline">
          <div>
            <NamespaceSelector data={appSelection} selectedNamespace={selectedNamespace} setSelectedNamespace={setSelectedNamespace} />
          </div>
          <div className="flex flex-row space-x-2 items-baseline">
            <LabelNamespaceSwitch enabled={selectedNamespace.labeled} setEnabled={() => {
              const newNamespace = { ...selectedNamespace };
              newNamespace.labeled = !newNamespace.labeled;
              if (newNamespace.labeled) {
                newNamespace.objects = newNamespace.objects.map((o) => {
                  return { ...o, labeled: false };
                });
              }
              setAppSelection({
                namespaces: appSelection.namespaces.map((ns) => {
                  if (ns.name === newNamespace.name) {
                    return newNamespace;
                  }
                  return ns;
                }),
              });
              setSelectedNamespace(newNamespace);
            }} />
            <div className="font-light">Select everything in this namespace</div>
          </div>
        </div>
      </div>
      <div className="pt-4">
        <AppsGrid
          selectedNamespace={selectedNamespace}
          changeObjectLabel={(obj) => {
            const newNamespace = { ...selectedNamespace };
            newNamespace.objects = newNamespace.objects.map((o) => {
              if (o.name === obj.name && o.kind === obj.kind) {
                return { ...o, labeled: !o.labeled };
              }
              return o;
            });
            setAppSelection({
              namespaces: appSelection.namespaces.map((ns) => {
                if (ns.name === newNamespace.name) {
                  return newNamespace;
                }
                return ns;
              }),
            });
            setSelectedNamespace(newNamespace);
          }}
        />
      </div>
      <button
        type="button"
        onClick={() => submitChanges(appSelection)}
        className="mt-6 text-white focus:ring-4 font-bold rounded-md text-sm px-14 py-2.5 mr-2 mb-2 bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-blue-800"
      >
        Save Changes
      </button>
    </div>
  );
};

function LabelNamespaceSwitch({ enabled, setEnabled}: any) {
  return (
    <Switch
      checked={enabled}
      onChange={setEnabled}
      className={`${enabled ? 'bg-blue-600' : 'bg-gray-200'
        } relative inline-flex h-6 w-11 items-center rounded-full`}
    >
      <span className="sr-only">Select everything in this namespace</span>
      <span
        className={`${enabled ? 'translate-x-6' : 'translate-x-1'
          } inline-block h-4 w-4 transform rounded-full bg-white transition`}
      />
    </Switch>
  )
}

export default SetupPage;
