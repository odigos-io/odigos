import { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import { KubernetesObjectsInNamespaces } from "@/types/apps";

export default async function persistConfiguration(
  req: NextApiRequest,
  res: NextApiResponse
) {
  const { data: KubernetesObjectsInNamespaces } = req.body;
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CoreV1Api);
  await k8sApi.listNamespace().then(async (response) => {
    response.body.items.forEach(async (item) => {
      const labeledReq = data.namespaces.find((ns) => ns.name === item.metadata?.name)?.labeled;
      const odigosLabeled = item.metadata?.labels?.["odigos.io/odigos-labeled"];
      if (labeledReq && odigosLabeled !== "true") {
        console.log("labeling namespace", item.metadata.name);
        item.metadata.labels = {
          ...item.metadata.labels,
          "odigos.io/odigos-labeled": "true",
        };
        await k8sApi.replaceNamespace(item.metadata.name, item);
      } else if (!labeledReq && odigosLabeled === "true") {
        console.log("unlabeling namespace", item.metadata.name);
        delete item.metadata.labels?.["odigos.io/odigos-labeled"];
        await k8sApi.replaceNamespace(item.metadata.name, item);
      }
    });
  });
  return res.status(200).end();
  // const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  // const response: any = await k8sApi.createNamespacedCustomObject(
  //   "odigos.io",
  //   "v1alpha1",
  //   process.env.CURRENT_NS || "odigos-system",
  //   "odigosconfigurations",
  //   {
  //     apiVersion: "odigos.io/v1alpha1",
  //     kind: "OdigosConfiguration",
  //     metadata: {
  //       name: "odigos-config",
  //     },
  //     spec: {
  //       instrumentationMode: req.body.instMode,
  //     },
  //   }
  // );

  // if (req.body.instMode === "OPT_IN" && req.body.selectedApps) {
  //   const instApps: any = await k8sApi.listClusterCustomObject(
  //     "odigos.io",
  //     "v1alpha1",
  //     "instrumentedapplications"
  //   );

  //   instApps.body.items
  //     .filter((item: any) => req.body.selectedApps.includes(item.metadata.uid))
  //     .map((item: any) => {
  //       item.spec.enabled = true;
  //       return item;
  //     })
  //     .forEach(async (item: any) => {
  //       await k8sApi.replaceNamespacedCustomObject(
  //         "odigos.io",
  //         "v1alpha1",
  //         item.metadata.namespace,
  //         "instrumentedapplications",
  //         item.metadata.name,
  //         item
  //       );
  //     });
  // }
}
