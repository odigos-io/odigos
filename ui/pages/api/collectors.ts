import type { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import type { IError } from "@/types/common";
import type { ICollectorsResponse } from "@/types/collectors";

async function GetCollectors(
  req: NextApiRequest,
  res: NextApiResponse<ICollectorsResponse | IError>
) {
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const kubeResp: any = await k8sApi.listNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "collectors"
  );

  const resp: ICollectorsResponse = {
    total: kubeResp.body.items.length,
    ready: kubeResp.body.items.filter((item: any) => item.status.ready).length,
  };
  return res.status(200).json(resp);
}

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<ICollectorsResponse | IError>
) {
  if (req.method === "GET") {
    return GetCollectors(req, res);
  }

  return res.status(405).end(`Method ${req.method} Not Allowed`);
}
