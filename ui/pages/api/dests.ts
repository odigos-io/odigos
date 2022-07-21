import type { NextApiRequest, NextApiResponse } from "next";
import * as k8s from "@kubernetes/client-node";
import type { DestResponseItem } from "@/types/dests";
import Vendors, { ObservabilityVendor, VendorObjects } from "@/vendors/index";

interface DestinationSecretRef {
  name: string;
}

interface DestinationSpec {
  type: string;
  data?: any;
  signals: string[];
  secretRef?: DestinationSecretRef;
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
  const vendor = Vendors.find(
    (v: ObservabilityVendor) => v.name === req.body.type
  );

  if (!vendor) {
    return res.status(400).json({
      error: `Vendor ${req.body.type} not found`,
    });
  }

  const kubeObjects: VendorObjects = vendor.toObjects(req);
  const selectedSignals = vendor.supportedSignals
    .filter((s: string) => req.body[s])
    .map((s: string) => s.toUpperCase());

  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const dest: Destination = {
    apiVersion: "odigos.io/v1alpha1",
    kind: "Destination",
    metadata: {
      name: req.body.name.toLowerCase(),
    },
    spec: {
      type: vendor.name,
      signals: selectedSignals,
    },
  };

  if (kubeObjects.Secret) {
    const secretName = await createSecretForDest(kc, req, kubeObjects.Secret);
    dest.spec!.secretRef = {
      name: secretName,
    };
  }

  if (kubeObjects.Data) {
    dest.spec!.data = {
      [vendor.name]: kubeObjects.Data,
    };
  } else {
    dest.spec!.data = {};
  }

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
  secretData: { [key: string]: string }
): Promise<string> {
  const k8sApi = kc.makeApiClient(k8s.CoreV1Api);
  const secret: k8s.V1Secret = {
    metadata: {
      name: req.body.name.toLowerCase(),
    },
    data: secretData,
  };

  const resp = await k8sApi.createNamespacedSecret(
    process.env.CURRENT_NS || "odigos-system",
    secret
  );

  return req.body.name.toLowerCase();
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
