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
      await updateDeployment(k8sApi, req.query.namespace as string, req.query.name as string, req.body.enabled, req.body.reportedName);
      break;
    case "statefulset":
      await updateStatefulSet(k8sApi, req.query.namespace as string, req.query.name as string, req.body.enabled, req.body.reportedName);
      break;
    case "daemonset":
      await updateDaemonSet(k8sApi, req.query.namespace as string, req.query.name as string, req.body.enabled, req.body.reportedName);
      break;
    default:
      return res.status(400).json({
        message: "kind is not supported",
      });
  }

  return res.status(200).json({
    message: "success",
  });
}

async function updateDeployment(k8sApi: k8s.AppsV1Api, namespace: string, name: string, enabled: boolean, reportedName: string) {
  const resp: any = await k8sApi.readNamespacedDeployment(name, namespace);
  resp.body.metadata.labels = resp.body.metadata.labels || {};
  resp.body.metadata.labels["odigos-instrumentation"] = enabled ? "enabled" : "disabled";
  resp.body.metadata.annotations = resp.body.metadata.annotations || {};
  resp.body.metadata.annotations["odigos.io/reported-name"] = reportedName;
  await k8sApi.replaceNamespacedDeployment(name, namespace, resp.body);
}

async function updateStatefulSet(k8sApi: k8s.AppsV1Api, namespace: string, name: string, enabled: boolean, reportedName: string) {
  const resp: any = await k8sApi.readNamespacedStatefulSet(name, namespace);
  resp.body.metadata.labels = resp.body.metadata.labels || {};
  resp.body.metadata.labels["odigos-instrumentation"] = enabled ? "enabled" : "disabled";
  resp.body.metadata.annotations = resp.body.metadata.annotations || {};
  resp.body.metadata.annotations["odigos.io/reported-name"] = reportedName;
  await k8sApi.replaceNamespacedStatefulSet(name, namespace, resp.body);
}

async function updateDaemonSet(k8sApi: k8s.AppsV1Api, namespace: string, name: string, enabled: boolean, reportedName: string) {
  const resp: any = await k8sApi.readNamespacedDaemonSet(name, namespace);
  resp.body.metadata.labels = resp.body.metadata.labels || {};
  resp.body.metadata.labels["odigos-instrumentation"] = enabled ? "enabled" : "disabled";
  resp.body.metadata.annotations = resp.body.metadata.annotations || {};
  resp.body.metadata.annotations["odigos.io/reported-name"] = reportedName;
  await k8sApi.replaceNamespacedDaemonSet(name, namespace, resp.body);
}