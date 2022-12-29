import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorObjects,
  VendorType,
} from "@/vendors/index";
import DatadogLogo from "@/img/vendor/datadog.svg";
import { NextApiRequest } from "next";

export class Datadog implements ObservabilityVendor {
  name = "datadog";
  displayName = "Datadog";
  type = VendorType.MANAGED;
  supportedSignals = [
    ObservabilitySignals.Traces,
    ObservabilitySignals.Metrics,
    ObservabilitySignals.Logs,
  ];

  getLogo = (props: any) => {
    return <DatadogLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
      {
        displayName: "Site",
        id: "site",
        name: "site",
        type: "text",
      },
      {
        displayName: "API Key",
        id: "apikey",
        name: "apikey",
        type: "password",
      },
    ];
  };

  toObjects = (req: NextApiRequest) => {
    return {
      Data: {
        DATADOG_SITE: req.body.site,
      },
      Secret: {
        DATADOG_API_KEY: Buffer.from(req.body.apikey).toString("base64"),
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      site: data.DATADOG_SITE,
    };
  };
}
