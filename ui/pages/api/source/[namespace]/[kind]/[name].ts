import { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";

export default async function UpdateSource(
  req: NextApiRequest,
  res: NextApiResponse
) {
  if (!req.query.kind || typeof req.query.kind !== "string") {
    return res.status(400).json({
      message: "kind is required",
    });
  }

  console.log(`updating source ${req.query.name} enabled: ${req.body.enabled}`);
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.AppsV1Api);
  switch (req.query.kind.toLowerCase()) {
    case "deployment":
      await updateDeployment(k8sApi, req.query.namespace as string, req.query.name as string, req.body.enabled);
      break;
    case "statefulset":
      await updateStatefulSet(k8sApi, req.query.namespace as string, req.query.name as string, req.body.enabled);
      break;
    case "daemonset":
      await updateDaemonSet(k8sApi, req.query.namespace as string, req.query.name as string, req.body.enabled);
      break;
    default:
      return res.status(400).json({
        message: "kind is not supported",
      });
  }

  return res.status(200).json({
    message: "success",
  });

  // const resp: any = await k8sApi.getNamespacedCustomObject(
  //   "odigos.io",
  //   "v1alpha1",
  //   req.query.namespace as string,
  //   "instrumentedapplications",
  //   `${req.query.kind}-${req.query.name}`
  // );

  // resp.body.spec.enabled = req.body.enabled;
  // await k8sApi.replaceNamespacedCustomObject(
  //   "odigos.io",
  //   "v1alpha1",
  //   req.query.namespace as string,
  //   "instrumentedapplications",
  //   `${req.query.kind}-${req.query.name}`,
  //   resp.body
  // );

  // res.status(200).json({ sucess: true });
}

async function updateDeployment(k8sApi: k8s.AppsV1Api, namespace: string, name: string, enabled: boolean) {
  const resp: any = await k8sApi.readNamespacedDeployment(name, namespace);
  resp.body.metadata.labels = resp.body.metadata.labels || {};
  resp.body.metadata.labels["odigos-instrumentation"] = enabled ? "enabled" : "disabled";
  await k8sApi.replaceNamespacedDeployment(name, namespace, resp.body);
}

async function updateStatefulSet(k8sApi: k8s.AppsV1Api, namespace: string, name: string, enabled: boolean) {
  const resp: any = await k8sApi.readNamespacedStatefulSet(name, namespace);
  resp.body.metadata.labels = resp.body.metadata.labels || {};
  resp.body.metadata.labels["odigos-instrumentation"] = enabled ? "enabled" : "disabled";
  await k8sApi.replaceNamespacedStatefulSet(name, namespace, resp.body);
}

async function updateDaemonSet(k8sApi: k8s.AppsV1Api, namespace: string, name: string, enabled: boolean) {
  const resp: any = await k8sApi.readNamespacedDaemonSet(name, namespace);
  resp.body.metadata.labels = resp.body.metadata.labels || {};
  resp.body.metadata.labels["odigos-instrumentation"] = enabled ? "enabled" : "disabled";
  await k8sApi.replaceNamespacedDaemonSet(name, namespace, resp.body);
}