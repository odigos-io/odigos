import { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";

export default async function persistConfiguration(
  req: NextApiRequest,
  res: NextApiResponse
) {
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const response: any = await k8sApi.createNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "odigosconfigurations",
    {
      apiVersion: "odigos.io/v1alpha1",
      kind: "OdigosConfiguration",
      metadata: {
        name: "odigos-config",
      },
      spec: {
        instrumentationMode: req.body.instMode,
      },
    }
  );

  if (req.body.instMode === "OPT_IN" && req.body.selectedApps) {
    const instApps: any = await k8sApi.listClusterCustomObject(
      "odigos.io",
      "v1alpha1",
      "instrumentedapplications"
    );

    instApps.body.items
      .filter((item: any) => req.body.selectedApps.includes(item.metadata.uid))
      .map((item: any) => {
        item.spec.enabled = true;
        return item;
      })
      .forEach(async (item: any) => {
        await k8sApi.replaceNamespacedCustomObject(
          "odigos.io",
          "v1alpha1",
          item.metadata.namespace,
          "instrumentedapplications",
          item.metadata.name,
          item
        );
      });
  }

  return res.status(200).end();
}
