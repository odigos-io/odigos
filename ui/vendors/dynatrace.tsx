import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorType,
} from "@/vendors/index";
import DynatraceLogo from "@/img/vendor/dynatrace.svg";
import { NextApiRequest } from "next";

export class Dynatrace implements ObservabilityVendor {
  name = "dynatrace";
  displayName = "dynatrace";
  type = VendorType.MANAGED;
  supportedSignals = [
      ObservabilitySignals.Traces,
      ObservabilitySignals.Metrics,
      ObservabilitySignals.Logs,
     ];
  getLogo = (props: any) => {
    return <DynatraceLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
      {
        displayName: "API Access Token",
        id: "dynatrace_access_token",
        name: "dynatrace_access_token",
        type: "password",
      },
      {
        displayName: "Dynatrace tenant url",
        id: "dynatrace_url",
        name: "dynatrace_url",
        type: "url",
      },
    ];
  };

  toObjects = (req: NextApiRequest) => {
    return {
      Data: {
        DYNATRACE_URL: req.body.dynatrace_url
      },
      Secret: {
          DYNATRACE_API_TOKEN: Buffer.from(req.body.dynatrace_access_token).toString(
          "base64"
        ),
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      dynatrace_url: data.DYNATRACE_URL || "",
    };
  };
}