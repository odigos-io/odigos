import { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import { KubernetesNamespace, KubernetesObjectsInNamespaces, AppKind } from "@/types/apps";

const odigosLabelKey = "odigos-instrumentation";
const odigosLabelValue = "enabled";

export default async function persistApplicationSelection(
  req: NextApiRequest,
  res: NextApiResponse
) {
  const data = req.body.data as KubernetesObjectsInNamespaces;
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CoreV1Api);
  await k8sApi.listNamespace().then(async (response) => {
    response.body.items.forEach(async (item) => {
      const kubeNamespace = data.namespaces.find((ns) => ns.name === item.metadata?.name);
      if (!kubeNamespace || !item.metadata?.name) {
        return;
      }

      const labeledReq = kubeNamespace?.labeled;
      const odigosLabeled = item.metadata?.labels?.[odigosLabelKey];
      if (labeledReq && odigosLabeled !== odigosLabelValue) {
        console.log("labeling namespace", item.metadata?.name);
        item.metadata.labels = {
          ...item.metadata?.labels,
          [odigosLabelKey]: odigosLabelValue,
        };
        await k8sApi.replaceNamespace(item.metadata.name, item);
      } else if (!labeledReq && odigosLabeled === odigosLabelValue) {
        console.log("unlabeling namespace", item.metadata?.name);
        delete item.metadata.labels?.[odigosLabelKey];
        await k8sApi.replaceNamespace(item.metadata.name, item);
      }
      await syncObjectsInNamespace(kc, kubeNamespace);
    });
  });

  return res.status(200).end();
}

async function syncObjectsInNamespace(kc: k8s.KubeConfig, ns: KubernetesNamespace) {
  const k8sApi = kc.makeApiClient(k8s.AppsV1Api);

  // Deployments
  await k8sApi.listNamespacedDeployment(ns.name).then(async (response) => {
    response.body.items.forEach(async (item) => {
      if (!item.metadata?.name) {
        return;
      }

      const labeledReq = ns.objects
        .find((d) => d.name === item.metadata?.name && d.kind.toString() === AppKind[AppKind.Deployment])
        ?.labeled;
      const odigosLabeled = item.metadata?.labels?.[odigosLabelKey];
      if (labeledReq && odigosLabeled !== odigosLabelValue) {
        console.log("labeling deployment", item.metadata?.name);
        item.metadata.labels = {
          ...item.metadata?.labels,
          [odigosLabelKey]: odigosLabelValue,
        };
        try {
          await k8sApi.replaceNamespacedDeployment(item.metadata.name, ns.name, item);
        } catch (e) {
          console.log(e);
        }
      } else if (!labeledReq && odigosLabeled === odigosLabelValue) {
        console.log("unlabeling deployment", item.metadata.name);
        delete item.metadata?.labels?.[odigosLabelKey];
        await k8sApi.replaceNamespacedDeployment(item.metadata.name, ns.name, item);
      }
    });
  });

  // StatefulSets
  await k8sApi.listNamespacedStatefulSet(ns.name).then(async (response) => {
    response.body.items.forEach(async (item) => {
      if (!item.metadata?.name) {
        return;
      }

      const labeledReq = ns.objects
        .find((d) => d.name === item.metadata?.name && d.kind.toString() === AppKind[AppKind.StatefulSet])
        ?.labeled;
      const odigosLabeled = item.metadata?.labels?.[odigosLabelKey];
      if (labeledReq && odigosLabeled !== odigosLabelValue) {
        console.log("labeling statefulset", item.metadata?.name);
        item.metadata.labels = {
          ...item.metadata?.labels,
          [odigosLabelKey]: odigosLabelValue,
        };
        await k8sApi.replaceNamespacedStatefulSet(item.metadata.name, ns.name, item);
      } else if (!labeledReq && odigosLabeled === odigosLabelValue) {
        console.log("unlabeling statefulset", item.metadata.name);
        delete item.metadata?.labels?.[odigosLabelKey];
        await k8sApi.replaceNamespacedStatefulSet(item.metadata.name, ns.name, item);
      }
    });
  });

  // DaemonSets
  await k8sApi.listNamespacedDaemonSet(ns.name).then(async (response) => {
    response.body.items.forEach(async (item) => {
      if (!item.metadata?.name) {
        return;
      }

      const labeledReq = ns.objects
        .find((d) => d.name === item.metadata?.name && d.kind.toString() === AppKind[AppKind.DaemonSet])
        ?.labeled;
      const odigosLabeled = item.metadata?.labels?.[odigosLabelKey];
      if (labeledReq && odigosLabeled !== odigosLabelValue) {
        console.log("labeling daemonset", item.metadata?.name);
        item.metadata.labels = {
          ...item.metadata?.labels,
          [odigosLabelKey]: odigosLabelValue,
        };
        await k8sApi.replaceNamespacedDaemonSet(item.metadata.name, ns.name, item);
      } else if (!labeledReq && odigosLabeled === odigosLabelValue) {
        console.log("unlabeling daemonset", item.metadata.name);
        delete item.metadata?.labels?.[odigosLabelKey];
        await k8sApi.replaceNamespacedDaemonSet(item.metadata.name, ns.name, item);
      }
    });
  });
}