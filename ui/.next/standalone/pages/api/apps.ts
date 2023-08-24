import type { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import type { AppsApiResponse, ApplicationData, KubernetesObjectsInNamespaces } from "@/types/apps";
import { GetAllKubernetesObjects } from "@/utils/kube";
type Error = {
  message: string;
};

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<AppsApiResponse | Error>
) {
  const kubeObjects = await GetAllKubernetesObjects();
  if (kubeObjects instanceof Error) {
    return res.status(500).json({
      message: kubeObjects.message,
    });
  }

  const enrichedApps = await enrichKubeObjectsWithRuntime(kubeObjects);
  if (enrichedApps instanceof Error) {
    return res.status(500).json({
      message: enrichedApps.message,
    });
  }

  return res.status(200).json({
    apps: enrichedApps as ApplicationData[],
  });
}

async function enrichKubeObjectsWithRuntime(kubeObjects: KubernetesObjectsInNamespaces): Promise<ApplicationData[] | Error> {
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const response: any = await k8sApi.listClusterCustomObject(
    "odigos.io",
    "v1alpha1",
    "instrumentedapplications"
  );

  const enrichedApps: ApplicationData[] = kubeObjects.namespaces.flatMap((ns) => {
    return ns.objects.map((obj) => {
      const app = response.body.items.find((item: any) => {
        return obj.kind.toString().toLowerCase() === item.metadata.ownerReferences[0].kind.toString().toLowerCase() &&
          obj.name.toLowerCase() === item.metadata.ownerReferences[0].name.toLowerCase();
      });

      if (!app) {
        return {
          id: `${obj.kind}-${obj.name}`,
          name: obj.name,
          namespace: ns.name,
          kind: obj.kind,
          instrumented: false,
          languages: [],
        };
      }

      return {
        id: app.metadata.uid,
        name: obj.name,
        namespace: ns.name,
        kind: obj.kind,
        instrumented: true,
        languages: app.spec.languages.map((lang: any) => lang.language),
      };
    });
  });

  return enrichedApps;
}