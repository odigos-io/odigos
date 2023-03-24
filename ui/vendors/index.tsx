import { Honeycomb } from "@/vendors/honeycomb";
import { Datadog } from "@/vendors/datadog";
import { Grafana } from "@/vendors/grafana";
import { NewRelic } from "@/vendors/newrelic";
import { Logzio } from "@/vendors/logzio";
import { Qryn } from "@/vendors/qryn";
import { Splunk } from "@/vendors/splunk";
import { Lightstep } from "@/vendors/lightstep";
import { NextApiRequest } from "next";
import { Prometheus } from "@/vendors/hosted/prometheus";
import { Tempo } from "@/vendors/hosted/tempo";
import { Loki } from "@/vendors/hosted/loki";
import { OpenTelemetry } from "@/vendors/hosted/opentelemetry";
import { Jaeger } from "@/vendors/hosted/jaeger";
import { SigNoz } from "@/vendors/hosted/signoz";
import { OpsVerse } from "@/vendors/opsverse";
import { GoogleCloud } from "@/vendors/googlecloud";
import { Elasticsearch } from '@/vendors/hosted/elasticsearch';
import { Sentry } from '@/vendors/sentry';
import { GoogleCloudStorage } from "@/vendors/gcs";
import { AzureBlobStorage } from "@/vendors/azureblobstorage";
import { AWSS3 } from "@/vendors/s3";

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
  new Logzio(),
  new Prometheus(),
  new Tempo(),
  new Loki(),
  new Jaeger(),
  new Elasticsearch(),
  new SigNoz(),
  new OpenTelemetry(),
  new Qryn(),
  new OpsVerse(),
  new Splunk(),
  new Lightstep(),
  new GoogleCloud(),
  new GoogleCloudStorage(),
  new AzureBlobStorage(),
  new AWSS3(),
  new Sentry(),
];

export default Vendors;
