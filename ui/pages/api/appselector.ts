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

  return res.status(200).end();
}
