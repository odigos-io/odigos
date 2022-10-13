import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorObjects,
  VendorType,
  IDestField,
} from "@/vendors/index";
import OpenTelemetryLogo from "@/img/vendor/opentelemetry.svg";
import { NextApiRequest } from "next";

export class OpenTelemetry implements ObservabilityVendor {
  name = "opentelemetry";
  displayName = "OpenTelemetry";
  type = VendorType.HOSTED;
  supportedSignals = [
    ObservabilitySignals.Traces,
    ObservabilitySignals.Metrics,
    ObservabilitySignals.Logs,
  ];
  getLogo = (props: any) => {
    return <OpenTelemetryLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
      {
        displayName: "OTLP URL",
        id: "otlp_url",
        name: "otlp_url",
        type: "url",
      },
    ];
  };

  toObjects = (req: NextApiRequest) => {
    return {
      Data: {
        OTLP_URL: req.body.otlp_url,
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      otlp_url: data.OTLP_URL || "",
    };
  };
}
