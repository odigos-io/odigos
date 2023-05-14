import type { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import type { AppsApiResponse, ApplicationData } from "@/types/apps";
import {stripPrefix} from "@/utils/crd";

type Error = {
  message: string;
};

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<AppsApiResponse | Error>
) {
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const response: any = await k8sApi.listClusterCustomObject(
    "odigos.io",
    "v1alpha1",
    "instrumentedapplications"
  );

  if (response.body.items.length === 0) {
    return res.status(404).json({
      message: "No apps found",
    });
  }

  const appsFound: ApplicationData[] = response.body.items
    .filter(
      (item: any) =>
        item.spec.languages &&
        item.spec.languages.length > 0 &&
        item.metadata.ownerReferences &&
        item.metadata.ownerReferences.length > 0
    )
    .map((item: any) => {
      const languages: string[] = item.spec.languages.map(
        (lang: any) => lang.language
      );

      return {
        id: item.metadata.uid,
        name: item.metadata.ownerReferences[0].name,
        languages: languages,
        instrumented: item.spec.languages.length > 0,
        kind: item.metadata.ownerReferences[0].kind,
        namespace: item.metadata.namespace,
      };
    });

  return res.status(200).json({
    apps: appsFound,
  });
}
