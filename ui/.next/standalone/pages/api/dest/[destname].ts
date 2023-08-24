import { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import { Socket } from "dgram";

async function UpdateDest(req: NextApiRequest, res: NextApiResponse) {
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const current = await k8sApi.getNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "destinations",
    req.query.destname as string
  );

  const updated = current.body;
  const { spec }: any = updated;
  spec.data = {
    [req.body.destType]: JSON.parse(req.body.values),
  };

  const resp = await k8sApi.replaceNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "destinations",
    req.query.destname as string,
    {
      ...updated,
      spec,
    }
  );

  return res.status(200).json({ success: true });
}

async function DeleteDest(req: NextApiRequest, res: NextApiResponse) {
  console.log(`Deleting destination ${req.query.destname}`);
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  await k8sApi.deleteNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "destinations",
    req.query.destname as string
  );

  // if secret with name req.query.destname exists, delete it
  const coreApi = kc.makeApiClient(k8s.CoreV1Api);
  const secret = await coreApi.readNamespacedSecret(
    req.query.destname as string,
    process.env.CURRENT_NS || "odigos-system"
  );

  if (secret) {
    await coreApi.deleteNamespacedSecret(
      req.query.destname as string,
      process.env.CURRENT_NS || "odigos-system"
    );
  }

  return res.status(200).json({ success: true });
}

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  if (req.method === "POST") {
    return UpdateDest(req, res);
  } else if (req.method === "DELETE") {
    return DeleteDest(req, res);
  }

  return res.status(405).end(`Method ${req.method} Not Allowed`);
}
