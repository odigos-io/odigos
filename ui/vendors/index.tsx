import { Honeycomb } from "@/vendors/honeycomb";
import { Datadog } from "@/vendors/datadog";
import { Grafana } from "@/vendors/grafana";
import { NewRelic } from "@/vendors/newrelic";
import { NextApiRequest } from "next";
import { Prometheus } from "@/vendors/hosted/prometheus";
import { Tempo } from "@/vendors/hosted/tempo";
import { Loki } from "@/vendors/hosted/loki";

export enum VendorType {
  MANAGED = "MANAGED",
  HOSTED = "HOSTED",
}

export enum ObservabilitySignals {
  Logs = "LOGS",
  Metrics = "METRICS",
  Traces = "TRACES",
}

export interface VendorObjects {
  Data?: { [key: string]: string };
  Secret?: { [key: string]: string };
}

export interface IDestField {
  displayName: string;
  id: string;
  name: string;
  type: string;
}

export interface ObservabilityVendor {
  name: string;
  displayName: string;
  supportedSignals: ObservabilitySignals[];
  type: VendorType;
  getLogo: (props: any) => any;
  getFields: (selectedSignals: any) => IDestField[];
  toObjects: (req: NextApiRequest) => VendorObjects;
  mapDataToFields: (data: any) => { [key: string]: string };
}

const Vendors = [
  new Honeycomb(),
  new Datadog(),
  new Grafana(),
  new NewRelic(),
  new Prometheus(),
  new Tempo(),
  new Loki(),
];

export default Vendors;
