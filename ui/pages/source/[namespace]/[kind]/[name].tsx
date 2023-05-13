import { useRouter } from "next/router";
import type { NextPage } from "next";
import * as k8s from "@kubernetes/client-node";
import { useState } from "react";

interface EditAppProps {
  enabled: boolean;
}

const EditAppPage: NextPage<EditAppProps> = ({ enabled }: EditAppProps) => {
  const router = useRouter();
  const { name, kind, namespace } = router.query;
  const [isEnabled, setIsEnabled] = useState(enabled);
  const updateApp = async () => {
    const resp = await fetch(`/api/source/${namespace}/${kind}/${name}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        enabled: isEnabled,
      }),
    });
    if (resp.ok) {
      router.push("/sources");
    }
  };
  return (
    <div className="flex flex-col w-fit">
      <div className="text-4xl font-medium">{name}</div>
      <label
        htmlFor="default-toggle"
        className="mt-12 inline-flex relative items-center cursor-pointer"
      >
        <input
          type="checkbox"
          value=""
          id="default-toggle"
          className="sr-only peer"
          onChange={() => {
            setIsEnabled(!isEnabled);
          }}
          checked={isEnabled}
        />
        <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
        <span className="ml-3 text-md font-medium text-gray-900">Enabled</span>
      </label>
      <button
        type="submit"
        disabled={isEnabled === enabled}
        onClick={updateApp}
        className="mt-4 disabled:cursor-not-allowed disabled:hover:bg-gray-500 disabled:bg-gray-500 text-white bg-blue-700 hover:bg-blue-800 focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 mr-2 mb-2 dark:bg-blue-600 dark:hover:bg-blue-700 focus:outline-none dark:focus:ring-blue-800"
      >
        Save Changes
      </button>
    </div>
  );
};

export const getServerSideProps = async ({ query }: any) => {
  const { name, kind, namespace } = query;
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const resp = await k8sApi.getNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    namespace,
    "instrumentedapplications",
    `${kind}-${name}`
  );

  if (!resp) {
    return {
      redirect: {
        destination: "/",
        permanent: false,
      },
    };
  }

  const { spec }: any = resp.body;
  // const isEnabled: boolean =
  //   config.spec.instrumentationMode === "OPT_IN"
  //     ? typeof spec.enabled === "boolean" && spec.enabled
  //     : spec.enabled === undefined || spec.enabled;
  return {
    props: {
      enabled: true,
    },
  };
};

export default EditAppPage;
