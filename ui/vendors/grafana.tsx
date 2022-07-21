import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorObjects,
} from "@/vendors/index";
import GrafanaLogo from "@/img/vendor/grafana.svg";
import { NextApiRequest } from "next";

export class Grafana implements ObservabilityVendor {
  name = "grafana";
  displayName = "Grafana Cloud";
  supportedSignals = [ObservabilitySignals.Traces];

  getLogo = (props: any) => {
    return <GrafanaLogo {...props} />;
  };

  getFields = () => {
    return [
      {
        displayName: "URL",
        id: "url",
        name: "url",
        type: "url",
      },
      {
        displayName: "User",
        id: "user",
        name: "user",
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
    // Grafana exporter expect token to be bas64 encoded, therefore we encode twice.
    const authString = Buffer.from(
      `${req.body.user}:${req.body.apikey}`
    ).toString("base64");

    return {
      Data: {
        url: req.body.url,
      },
      Secret: {
        AUTH_TOKEN: Buffer.from(authString).toString("base64"),
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      url: data.url,
    };
  };
}
