import {
    ObservabilityVendor,
    ObservabilitySignals,
    VendorObjects,
    VendorType,
    IDestField,
  } from "@/vendors/index";
  import SigNozLogo from "@/img/vendor/signoz.svg";
  import { NextApiRequest } from "next";
  
  export class SigNoz implements ObservabilityVendor {
    name = "signoz";
    displayName = "SigNoz";
    type = VendorType.HOSTED;
    supportedSignals = [
      ObservabilitySignals.Traces,
      ObservabilitySignals.Metrics,
      ObservabilitySignals.Logs,
    ];
    getLogo = (props: any) => {
      return <SigNozLogo {...props} />;
    };
  
    getFields = (selectedSignals: any) => {
      return [
        {
          displayName: "OpenTelemetry Collector URL",
          id: "signoz_url",
          name: "signoz_url",
          type: "url",
        },
      ];
    };
  
    toObjects = (req: NextApiRequest) => {
      return {
        Data: {
          SIGNOZ_URL: req.body.signoz_url,
        },
      };
    };
  
    mapDataToFields = (data: any) => {
      return {
        signoz_url: data.SIGNOZ_URL || "",
      };
    };
  }
  