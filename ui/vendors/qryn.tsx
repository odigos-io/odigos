import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorObjects,
  VendorType,
} from "@/vendors/index";
import QrynLogo from "@/img/vendor/qryn.svg";
import { NextApiRequest } from "next";

export class Qryn implements ObservabilityVendor {
  name = "qryn";
  displayName = "qryn";
  type = VendorType.MANAGED;
  supportedSignals = [
    ObservabilitySignals.Traces,
    ObservabilitySignals.Metrics,
    ObservabilitySignals.Logs,
  ];

  getLogo = (props: any) => {
    return <QrynLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    let fields = [
      {
        displayName: "qryn API Key",
        id: "apikey",
        name: "apikey",
        type: "password",
      },
      {
        displayName: "qryn API URL",
        id: "url",
        name: "url",
        type: "url",
      },
      {
        displayName: "qryn API User",
        id: "user",
        name: "user",
        type: "text",
      }
    ];

    return fields;
  };

  toObjects = (req: NextApiRequest) => {
    // Tempo exporter expect token to be bas64 encoded, therefore we encode twice.
    const authString = Buffer.from(
      `${req.body.tempo_user}:${req.body.apikey}`
    ).toString("base64");

    return {
      Data: {
        QRYN_URL: req.body.url,
        QRYN_USER: req.body.user,
      },
      Secret: {
        QRYN_TOKEN: Buffer.from(req.body.apikey).toString("base64"),
        QRYN_AUTH_TOKEN: Buffer.from(authString).toString("base64"),
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      url: data.QRYN_URL || "",
    };
  };
}
