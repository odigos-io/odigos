import type { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import type { DestResponseItem } from "@/types/dests";

enum DestinationType {
  Grafana = "grafana",
  Datadog = "datadog",
  Honeycomb = "honeycomb",
}

interface DestinationSpec {
  type: DestinationType;
  data: DestinationData;
}

interface DestinationData {
  grafana?: GrafanaData;
  honeycomb?: HoneycombData;
}

interface GrafanaData {
  url: string;
  user: string;
  apiKey: string;
}

interface HoneycombData {
  apiKey: string;
}

interface DestinationStatus {}

interface Destination {
  apiVersion: string;
  kind: string;
  metadata: k8s.V1ObjectMeta;
  spec?: DestinationSpec;
  status?: DestinationStatus;
}

async function CreateNewDestination(
  req: NextApiRequest,
  res: NextApiResponse<any>
) {
  const { type } = req.body;

  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);

  try {
    const dest: Destination = {
      apiVersion: "observability.control.plane.keyval.dev/v1alpha1",
      kind: "Destination",
      metadata: {
        name: req.body.name.toLowerCase(),
      },
      spec: {
        type: DestinationType.Grafana,
        data: {
          grafana: {
            apiKey: req.body.apikey,
            url: req.body.url,
            user: req.body.user,
          },
        },
      },
    };

    const resp = await k8sApi.createNamespacedCustomObject(
      "observability.control.plane.keyval.dev",
      "v1alpha1",
      process.env.CURRENT_NAMESPACE || "odigos-system",
      "destinations",
      dest
    );
  } catch (ex) {
    console.log(`got error: ${JSON.stringify(ex)}`);
  }

  res.redirect("/");
}

async function GetDestinations(req: NextApiRequest, res: NextApiResponse<any>) {
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);

  const response: any = await k8sApi.listNamespacedCustomObject(
    "observability.control.plane.keyval.dev",
    "v1alpha1",
    process.env.CURRENT_NAMESPACE || "odigos-system",
    "destinations"
  );

  if (response.body.items.length === 0) {
    res.status(404).json({
      message: "No dests found",
    });
  }

  const dests: DestResponseItem[] = response.body.items.map((item: any) => {
    return {
      id: item.metadata.uid,
      name: item.metadata.name,
      type: item.spec.type,
    };
  });

  return res.status(200).json(dests);
}

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<any>
) {
  if (req.method === "POST") {
    return CreateNewDestination(req, res);
  } else if (req.method === "GET") {
    return GetDestinations(req, res);
  }

  res.status(405).end(`Method ${req.method} Not Allowed`);
}
