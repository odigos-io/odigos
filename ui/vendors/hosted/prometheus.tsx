import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorObjects,
  VendorType,
} from "@/vendors/index";
import PrometheusLogo from "@/img/vendor/prometheus.svg";
import { NextApiRequest } from "next";

export class Prometheus implements ObservabilityVendor {
  name = "prometheus";
  displayName = "Prometheus";
  type = VendorType.HOSTED;
  supportedSignals = [ObservabilitySignals.Metrics];

  getLogo = (props: any) => {
    return <PrometheusLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
      {
        displayName: "Remote Write URL",
        id: "remotewrite_url",
        name: "remotewrite_url",
        type: "url",
      },
    ];
  };

  toObjects = (req: NextApiRequest) => {
    return {
      Data: {
        PROMETHEUS_REMOTEWRITE_URL: req.body.remotewrite_url,
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      remotewrite_url: data.PROMETHEUS_REMOTEWRITE_URL || "",
    };
  };
}
