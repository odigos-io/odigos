import type { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import type { ApplicationData } from "@/types/apps";

type Error = {
  message: string;
};

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<ApplicationData[] | Error>
) {
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);

  const response: any = await k8sApi.listNamespacedCustomObject(
    "observability.control.plane.keyval.dev",
    "v1",
    process.env.CURRENT_NAMESPACE || "odigos-system",
    "instrumentedapplications"
  );

  if (response.body.items.length === 0) {
    res.status(404).json({
      message: "No apps found",
    });
  }

  const apps: ApplicationData[] = response.body.items
    .filter((item: any) => item.spec.languages)
    .map((item: any) => {
      const languages: string[] = item.spec.languages.map(
        (lang: any) => lang.language
      );

      return {
        id: item.metadata.uid,
        name: item.spec.ref.name,
        languages: languages,
        instrumented: item.spec.instrumented,
      };
    });

  return res.status(200).json(apps);
}
