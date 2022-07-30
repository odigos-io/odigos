import {
  IDestField,
  ObservabilitySignals,
  ObservabilityVendor,
  VendorObjects,
} from "@/vendors/index";
import NewRelicLogo from "@/img/vendor/newrelic.svg";
import { NextApiRequest } from "next";

export class NewRelic implements ObservabilityVendor {
  name = "newrelic";
  displayName = "New Relic";
  supportedSignals = [
    ObservabilitySignals.Traces,
    ObservabilitySignals.Metrics,
    ObservabilitySignals.Logs,
  ];

  getLogo = (props: any) => {
    return <NewRelicLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
      {
        displayName: "License Key",
        id: "apikey",
        name: "apikey",
        type: "password",
      },
    ];
  };

  toObjects = (req: NextApiRequest) => {
    return {
      Secret: {
        NEWRELIC_API_KEY: Buffer.from(req.body.apikey).toString("base64"),
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {};
  };
}
