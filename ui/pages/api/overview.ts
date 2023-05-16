import type { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import type { OverviewApiResponse } from "@/types/overview";
import { ApplicationData } from "@/types/apps";
import { Collector } from "@/types/collectors";
import { OverviewDestResponseItem } from "@/types/dests";

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<OverviewApiResponse>
) {
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const instAppsResponse: any = await k8sApi.listClusterCustomObject(
    "odigos.io",
    "v1alpha1",
    "instrumentedapplications"
  );

  const appsFound: ApplicationData[] = instAppsResponse.body.items.map(
    (item: any) => {
      return {
        id: item.metadata.uid,
        name: item.metadata.ownerReferences[0].name,
        languages: item.spec.languages?.map((lang: any) => lang.language),
        instrumented: item.spec.languages?.length > 0,
        kind: item.metadata.ownerReferences[0].kind,
        namespace: item.metadata.namespace,
      };
    }
  );

  const collectorsResp: any = await k8sApi.listNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "collectorsgroups"
  );

  const collectors: Collector[] = collectorsResp.body.items.map((item: any) => {
    return {
      name: item.metadata.name,
      ready: item.status.ready,
    };
  });

  const destResp: any = await k8sApi.listNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "destinations"
  );

  const dests: OverviewDestResponseItem[] = destResp.body.items.map(
    (item: any) => {
      return {
        id: item.metadata.uid,
        name: item.metadata.name,
        type: item.spec.type,
        signals: item.spec.signals,
      };
    }
  );

  return res.status(200).json({
    sources: appsFound,
    collectors: collectors,
    dests: dests,
  });
}
