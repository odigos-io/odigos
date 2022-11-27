import {
  ObservabilitySignals,
  ObservabilityVendor,
  VendorObjects,
  VendorType,
} from "@/vendors/index";
import HoneycombLogo from "@/img/vendor/honeycomb.svg";
import { NextApiRequest } from "next";

export class Honeycomb implements ObservabilityVendor {
  name = "honeycomb";
  displayName = "Honeycomb";
  type = VendorType.MANAGED;
  supportedSignals = [
    ObservabilitySignals.Traces,
    ObservabilitySignals.Metrics,
    ObservabilitySignals.Logs,
  ];

  getLogo = (props: any) => {
    return <HoneycombLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
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
      Secret: {
        HONEYCOMB_API_KEY: Buffer.from(req.body.apikey).toString("base64"),
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {};
  };
}
