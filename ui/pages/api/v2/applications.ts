import type { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import { KubernetesObjectsInNamespaces, KubernetesNamespace, KubernetesObject, AppKind } from "@/types/apps";
import { stripPrefix } from "@/utils/crd";

type Error = {
    message: string;
};

export default async function handler(
    req: NextApiRequest,
    res: NextApiResponse<KubernetesObjectsInNamespaces | Error>
) {
    const kc = new k8s.KubeConfig();
    kc.loadFromDefault();
    const k8sApi = kc.makeApiClient(k8s.CoreV1Api);
    const namespacesResponse: any = await k8sApi.listNamespace();
    const objectsByNamespace: any = {};
    for (const namespace of namespacesResponse.body.items) {
        const objectsInNamespace = await getObjectsInNamespace(namespace.metadata.name, kc);
        objectsByNamespace[namespace.metadata.name] = objectsInNamespace;
    }

    const namespaces: KubernetesNamespace[] = namespacesResponse.body.items.map((item: any) => {
        return {
            name: item.metadata.name,
            labeled: item.metadata.labels && item.metadata.labels["odigos-instrumentation"] === "enabled",
            objects: objectsByNamespace[item.metadata.name],
        };
    });

    return res.status(200).json({
        namespaces: namespaces,
    });
}

async function getObjectsInNamespace(namespace: string, kc: k8s.KubeConfig): Promise<KubernetesObject[]> {
    // Get deployments, statefulsets and daemonsets
    const k8sApi = kc.makeApiClient(k8s.AppsV1Api);
    const deploymentsResponse: any = await k8sApi.listNamespacedDeployment(namespace);
    const deployments: KubernetesObject[] = deploymentsResponse.body.items.map((item: any) => {
        return {
            name: item.metadata.name,
            kind: AppKind[AppKind.Deployment],
            instances: item.status.availableReplicas,
            labeled: item.metadata.labels && item.metadata.labels["odigos-instrumentation"] === "enabled",
        };
    }
    );

    const statefulsetsResponse: any = await k8sApi.listNamespacedStatefulSet(namespace);
    const statefulsets: KubernetesObject[] = statefulsetsResponse.body.items.map((item: any) => {
        return {
            name: item.metadata.name,
            kind: AppKind[AppKind.StatefulSet],
            instances: item.status.readyReplicas,
            labeled: item.metadata.labels && item.metadata.labels["odigos-instrumentation"] === "enabled",
        };
    }
    );

    const daemonsetsResponse: any = await k8sApi.listNamespacedDaemonSet(namespace);
    const daemonsets: KubernetesObject[] = daemonsetsResponse.body.items.map((item: any) => {
        return {
            name: item.metadata.name,
            kind: AppKind[AppKind.DaemonSet],
            instances: item.status.numberReady,
            labeled: item.metadata.labels && item.metadata.labels["odigos-instrumentation"] === "enabled",
        };
    });

    return deployments.concat(statefulsets).concat(daemonsets);
}
