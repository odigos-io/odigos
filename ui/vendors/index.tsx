import { Honeycomb } from "@/vendors/honeycomb";
import { Datadog } from "@/vendors/datadog";
import { Grafana } from "@/vendors/grafana";
import { NextApiRequest } from "next";

export enum ObservabilitySignals {
  Logs = "Logs",
  Metrics = "Metrics",
  Traces = "Traces",
}

export interface VendorObjects {
  Data?: any;
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
  getLogo: (props: any) => any;
  getFields: () => IDestField[];
  toObjects: (req: NextApiRequest) => VendorObjects;
  fromObjects: (vendorObjects: VendorObjects) => any;
}

const Vendors = [new Honeycomb(), new Datadog(), new Grafana()];

export default Vendors;
