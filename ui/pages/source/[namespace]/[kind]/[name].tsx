import { useRouter } from "next/router";
import type { NextPage } from "next";
import * as k8s from "@kubernetes/client-node";
import { useState } from "react";

interface EditAppProps {
  enabled: boolean;
  reportedName: string;
}

const EditAppPage: NextPage<EditAppProps> = ({ enabled, reportedName }: EditAppProps) => {
  const router = useRouter();
  const { name, kind, namespace } = router.query;
  const [isEnabled, setIsEnabled] = useState(enabled);
  const [updatedReportedName, setUpdatedReportedName] = useState(reportedName);
  const updateApp = async () => {
    const resp = await fetch(`/api/source/${namespace}/${kind}/${name}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        enabled: isEnabled,
        reportedName: updatedReportedName,
      }),
    });
    if (resp.ok) {
      router.push("/sources");
    }
  };
  return (
    <div className="flex flex-col w-fit">
      <div className="text-4xl font-medium">{name}</div>
      <div>
        <label className="block mt-6">
                    <span className="text-gray-700">Reported Name</span>
                    <input
                      name="reportedName"
                      type="text"
                      className="
                    mt-1
                    block
                    w-full
                    rounded-md
                    border-gray-300
                    shadow-sm
                    focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50
                  "
                      placeholder=""
                      required
                      defaultValue={reportedName}
                      onChange={(e) => {
                        setUpdatedReportedName(e.target.value);
                      }}
                    />
                  </label>
        </div>
      <label
        htmlFor="default-toggle"
        className="mt-6 inline-flex relative items-center cursor-pointer"
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
        disabled={isEnabled === enabled && reportedName === updatedReportedName}
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
  const k8sApi = kc.makeApiClient(k8s.AppsV1Api);
  const reportedNameAnootation = "odigos.io/reported-name";
  var obj = null;
  var instrumented = false;
  var reportedName = name;
  switch (kind) {
    case "deployment":
      obj = await getDeployment(name, namespace, kc);
      break;
    case "statefulset":
      obj = await getStatefulSet(name, namespace, kc);
      break;
    case "daemonset":
      obj = await getDaemonSet(name, namespace, kc);
      break;
    default: 
      return {
        redirect: {
          destination: "/",
          permanent: false,
        },
      };
  }

  if (!obj) {
    return {
      redirect: {
        destination: "/",
        permanent: false,
      },
    }
  }

  if (obj?.metadata?.annotations?.[reportedNameAnootation]) {
    reportedName = obj?.metadata?.annotations?.[reportedNameAnootation];
  }

  instrumented = isLabeled(obj?.metadata?.labels);
  if (!instrumented) {
    instrumented = await isNamespaceLabeled(namespace, kc);
  }

  return {
    props: {
      enabled: instrumented,
      reportedName: reportedName,
    },
  };
};

async function getDeployment(name: string, namespace: string, kc: k8s.KubeConfig) {
  const kubeClient = kc.makeApiClient(k8s.AppsV1Api);
  const resp = await kubeClient.readNamespacedDeployment(name, namespace);
  if (!resp) {
    return null;
  }

  return resp.body;
}

async function getStatefulSet(name: string, namespace: string, kc: k8s.KubeConfig) {
  const kubeClient = kc.makeApiClient(k8s.AppsV1Api);
  const resp = await kubeClient.readNamespacedStatefulSet(name, namespace);
  if (!resp) {
    return null;
  }

  return resp.body;
}

async function getDaemonSet(name: string, namespace: string, kc: k8s.KubeConfig) {
  const kubeClient = kc.makeApiClient(k8s.AppsV1Api);
  const resp = await kubeClient.readNamespacedDaemonSet(name, namespace);
  if (!resp) {
    return null;
  }

  return resp.body;
}

function isLabeled(labels: any): boolean {
  return labels && labels["odigos-instrumentation"] === "enabled";
}

async function isNamespaceLabeled(name: string, kc: k8s.KubeConfig) {
  const kubeClient = kc.makeApiClient(k8s.CoreV1Api);
  const resp = await kubeClient.readNamespace(name);
  if (!resp || !resp.body.metadata) {
    return false;
  }

  return isLabeled(resp.body.metadata.labels);
}

export default EditAppPage;
