import type { NextApiRequest, NextApiResponse } from "next";
import { KubernetesObjectsInNamespaces } from "@/types/apps";
import { GetAllKubernetesObjects } from "@/utils/kube";

type Error = {
    message: string;
};

export default async function handler(
    req: NextApiRequest,
    res: NextApiResponse<KubernetesObjectsInNamespaces | Error>
) {
    return res.status(200).json(await GetAllKubernetesObjects());
}