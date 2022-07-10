import type { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import type { IError } from "@/types/common";
import type { ICollectorsResponse } from "@/types/collectors";

async function DeleteCollector(req: NextApiRequest, res: NextApiResponse) {
  console.log(`deleting collector ${req.body.name}`);
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  await k8sApi.deleteNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "collectors",
    req.body.name as string
  );
  return res.status(200).json({ success: true });
}

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
    collectors: kubeResp.body.items.map((item: any) => {
      return {
        name: item.metadata.name,
        ready: item.status.ready,
      };
    }),
  };
  return res.status(200).json(resp);
}

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<ICollectorsResponse | IError>
) {
  if (req.method === "GET") {
    return GetCollectors(req, res);
  } else if (req.method === "DELETE") {
    return DeleteCollector(req, res);
  }

  return res.status(405).end(`Method ${req.method} Not Allowed`);
}
