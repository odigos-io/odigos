import type { NextPage } from "next";
import useSWR, { Fetcher } from "swr";
import { OverviewApiResponse } from "@/types/overview";
import LoadingPage from "@/components/Loading";
import { ApplicationData } from "@/types/apps";
import { getLangIcon } from "@/utils/icons";
import Vendors from "@/vendors/index";
import Link from "next/link";
import * as k8s from "@kubernetes/client-node";

const Home: NextPage = () => {
  const fetcher: Fetcher<OverviewApiResponse, any> = (args: any) =>
    fetch(args).then((res) => res.json());
  const { data, error } = useSWR<OverviewApiResponse>("/api/overview", fetcher);
  if (error) return <div>failed to load</div>;
  if (!data) return <LoadingPage />;
  const appsByLang = data.sources.reduce((acc: any, app: ApplicationData) => {
    const lang = app.languages ? app.languages[0] : "unrecognized";
    if (!acc[lang]) {
      acc[lang] = [];
    }
    acc[lang].push(app);
    return acc;
  }, {});

  const totalCollectors = data.collectors.length;
  const readyCollectors = data.collectors.filter((c) => c.ready).length;
  return (
    <div className="w-full h-full">
      <div className="h-1/3 w-full">
        <div className="text-4xl font-medium">Sources</div>
        <div className="mt-4 grid grid-flow-col grid-rows-4 gap-x-10 gap-y-2 w-fit">
          {appsByLang["unrecognized"]?.length > 0 && (
            <div className="flex flex-row items-center space-x-2">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="h-8 w-8 text-red-500"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fillRule="evenodd"
                  d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
                  clipRule="evenodd"
                />
              </svg>
              <div className="font-bold">
                {appsByLang["unrecognized"].length} unrecognized applications
              </div>
            </div>
          )}
          {Object.keys(appsByLang)
            .filter((lang) => lang !== "unrecognized")
            .map((lang) => (
              <div key={lang} className="flex flex-row items-center space-x-2">
                <div>{getLangIcon(lang, "w-8 h-8")}</div>
                <div className="">
                  <span className="text-bold text-2xl">
                    {appsByLang[lang].length}
                  </span>{" "}
                  {lang} applications
                </div>
              </div>
            ))}
        </div>
      </div>
      <div className="h-1/3 w-full">
        <div className="text-4xl font-medium">Destinations</div>
        {data.dests && data.dests.length > 0 ? (
          <div className="mt-4 grid grid-flow-col grid-rows-4 gap-x-10 gap-y-2 w-fit">
            {data.dests.map((dest) => (
              <div
                key={dest.id}
                className="flex flex-row items-center space-x-2"
              >
                {Vendors.find((v) => v.name === dest.type)?.getLogo({
                  className: "w-8 h-8",
                })}
                <div className="">
                  Sending{" "}
                  <span>
                    {dest.signals.map((s) => s.toLowerCase()).join(", ")}
                  </span>{" "}
                  to <span className="font-bold">{dest.name}</span>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div
            className="mt-4 w-fit flex p-4 mb-4 text-sm rounded-lg bg-yellow-200 text-yellow-700"
            role="alert"
          >
            <svg
              className="inline flex-shrink-0 mr-3 w-5 h-5"
              fill="currentColor"
              viewBox="0 0 20 20"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                fill-rule="evenodd"
                d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                clip-rule="evenodd"
              ></path>
            </svg>
            <div>
              <span className="font-medium">No destinations configured!</span>{" "}
              <Link href="/dest/new" className="font-medium underline">
                Click here to add a destination
              </Link>
            </div>
          </div>
        )}
      </div>
      <div className="h-1/3 w-full">
        <div className="text-4xl font-medium">Collectors</div>
        {totalCollectors > 0 ? (
          <div className="mt-4 flex flex-row space-x-2 items-center">
            {readyCollectors === totalCollectors ? (
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="h-8 w-8 text-green-600"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fillRule="evenodd"
                  d="M6.267 3.455a3.066 3.066 0 001.745-.723 3.066 3.066 0 013.976 0 3.066 3.066 0 001.745.723 3.066 3.066 0 012.812 2.812c.051.643.304 1.254.723 1.745a3.066 3.066 0 010 3.976 3.066 3.066 0 00-.723 1.745 3.066 3.066 0 01-2.812 2.812 3.066 3.066 0 00-1.745.723 3.066 3.066 0 01-3.976 0 3.066 3.066 0 00-1.745-.723 3.066 3.066 0 01-2.812-2.812 3.066 3.066 0 00-.723-1.745 3.066 3.066 0 010-3.976 3.066 3.066 0 00.723-1.745 3.066 3.066 0 012.812-2.812zm7.44 5.252a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                  clipRule="evenodd"
                />
              </svg>
            ) : (
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className="h-8 w-8 text-red-500"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fillRule="evenodd"
                  d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
                  clipRule="evenodd"
                />
              </svg>
            )}
            <div className="font-bold">
              {readyCollectors} / {totalCollectors} collectors ready
            </div>
          </div>
        ) : (
          <div
            className="w-fit mt-4 flex p-4 mb-4 text-sm rounded-lg bg-blue-200 text-blue-800"
            role="alert"
          >
            <svg
              className="inline flex-shrink-0 mr-3 w-5 h-5"
              fill="currentColor"
              viewBox="0 0 20 20"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                fill-rule="evenodd"
                d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                clip-rule="evenodd"
              ></path>
            </svg>
            <div className="font-medium max-w-md">
              No collectors running. Odigos will automaticly deploy collectors
              once a destination is configured.
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export const getServerSideProps = async ({ query }: any) => {
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const foundLabeled = await isSomethingLabeled(kc);
  if (!foundLabeled) {
    return {
      redirect: {
        destination: "/setup",
        permanent: false,
      },
    };
  }

  // Check if any destination is configured
  const kubeCrdApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const destinations: any = await kubeCrdApi.listNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "destinations"
  );

  if (destinations.body.items && destinations.body.items.length === 0) {
    return {
      redirect: {
        destination: "/dest/new",
        permanent: false,
      },
    };
  }

  return {
    props: {},
  };
};

async function isSomethingLabeled(kc: k8s.KubeConfig): Promise<boolean> {
  // Check if there is any namespace labeled with odigos
  const k8sApi = kc.makeApiClient(k8s.CoreV1Api);
  const namespaces = await k8sApi.listNamespace();
  const odigosNamespaces = namespaces.body.items.filter((ns) => {
    return (
      ns.metadata?.labels &&
      ns.metadata?.labels["odigos-instrumentation"] === "enabled"
    );
  });

  if (odigosNamespaces.length > 0) {
    return true;
  }

  // Check if there is any deployment labeled with odigos
  const k8sAppsApi = kc.makeApiClient(k8s.AppsV1Api);
  const deployments = await k8sAppsApi.listDeploymentForAllNamespaces();
  const odigosDeployments = deployments.body.items.filter((d) => {
    return (
      d.metadata?.labels &&
      d.metadata?.labels["odigos-instrumentation"] === "enabled"
    );
  });

  if (odigosDeployments.length > 0) {
    return true;
  }

  // Check if there is any daemonset labeled with odigos
  const daemonsets = await k8sAppsApi.listDaemonSetForAllNamespaces();
  const odigosDaemonsets = daemonsets.body.items.filter((d) => {
    return (
      d.metadata?.labels &&
      d.metadata?.labels["odigos-instrumentation"] === "enabled"
    );
  });

  if (odigosDaemonsets.length > 0) {
    return true;
  }

  // Check if there is any statefulset labeled with odigos
  const statefulsets = await k8sAppsApi.listStatefulSetForAllNamespaces();
  const odigosStatefulsets = statefulsets.body.items.filter((d) => {
    return (
      d.metadata?.labels &&
      d.metadata?.labels["odigos-instrumentation"] === "enabled"
    );
  });

  if (odigosStatefulsets.length > 0) {
    return true;
  }

  return false;
}

export default Home;
