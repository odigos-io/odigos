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
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const destName = req.body.type;
  const secretName = await createSecretForDest(kc, req, destName);
  const dest: Destination = {
    apiVersion: "odigos.io/v1alpha1",
    kind: "Destination",
    metadata: {
      name: req.body.name.toLowerCase(),
    },
    spec: {
      ...getSpecForDest(req, destName),
      secretRef: {
        name: secretName,
      },
    },
  };

  const resp = await k8sApi.createNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "destinations",
    dest
  );

  return res.status(200).json({ message: "dest created" });
}

async function createSecretForDest(
  kc: k8s.KubeConfig,
  req: NextApiRequest,
  destName: string
): Promise<string> {
  const k8sApi = kc.makeApiClient(k8s.CoreV1Api);
  const secret: k8s.V1Secret = {
    metadata: {
      name: req.body.name.toLowerCase(),
    },
    data: getSecretForDest(req, destName),
  };

  const resp = await k8sApi.createNamespacedSecret(
    process.env.CURRENT_NS || "odigos-system",
    secret
  );

  return req.body.name.toLowerCase();
}

function getSecretForDest(
  req: NextApiRequest,
  destName: string
): { [key: string]: string } {
  switch (destName) {
    case "honeycomb": {
      return {
        API_KEY: Buffer.from(req.body.apikey).toString("base64"),
      };
    }
    case "datadog": {
      return {
        API_KEY: Buffer.from(req.body.apikey).toString("base64"),
      };
    }
    case "grafana": {
      // Grafana exporter expect token to be bas64 encoded, therefore we encode twice.
      const authString = Buffer.from(
        `${req.body.user}:${req.body.apikey}`
      ).toString("base64");
      return {
        AUTH_TOKEN: Buffer.from(authString).toString("base64"),
      };
    }
  }

  throw new TypeError("unrecognized destination");
}

function getSpecForDest(req: NextApiRequest, destName: string): any {
  switch (destName) {
    case "grafana":
      return {
        type: DestinationType.Grafana,
        data: {
          grafana: {
            url: req.body.url,
          },
        },
      };
    case "honeycomb":
      return {
        type: DestinationType.Honeycomb,
        data: {},
      };
    case "datadog":
      return {
        type: DestinationType.Datadog,
        data: {
          datadog: {
            site: req.body.site,
          },
        },
      };
  }

  return null;
}

async function GetDestinations(req: NextApiRequest, res: NextApiResponse<any>) {
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);

  const response: any = await k8sApi.listNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "destinations"
  );

  if (response.body.items.length === 0) {
    return res.status(404).json({
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

  return res.status(405).end(`Method ${req.method} Not Allowed`);
}
