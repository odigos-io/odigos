import { Honeycomb } from "@/vendors/honeycomb";
import { Datadog } from "@/vendors/datadog";
import { Grafana } from "@/vendors/grafana";
import { NewRelic } from "@/vendors/newrelic";
import { NextApiRequest } from "next";

export enum ObservabilitySignals {
  Logs = "Logs",
  Metrics = "Metrics",
  Traces = "Traces",
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
  getLogo: (props: any) => any;
  getFields: (selectedSignals: any) => IDestField[];
  toObjects: (req: NextApiRequest) => VendorObjects;
  mapDataToFields: (data: any) => { [key: string]: string };
}

const Vendors = [new Honeycomb(), new Datadog(), new Grafana(), new NewRelic()];

export default Vendors;
