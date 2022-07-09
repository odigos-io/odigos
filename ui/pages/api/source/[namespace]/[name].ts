import { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";

export default async function UpdateSource(
  req: NextApiRequest,
  res: NextApiResponse
) {
  console.log(`updating source ${req.query.name} enabled: ${req.body.enabled}`);
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const resp: any = await k8sApi.getNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    req.query.namespace as string,
    "instrumentedapplications",
    req.query.name as string
  );

  resp.body.spec.enabled = req.body.enabled;
  await k8sApi.replaceNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    req.query.namespace as string,
    "instrumentedapplications",
    req.query.name as string,
    resp.body
  );

  res.status(200).json({ sucess: true });
}
