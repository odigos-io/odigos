import {
    ObservabilityVendor,
    ObservabilitySignals,
    VendorType,
  } from "@/vendors/index";
  import SentryLogo from "@/img/vendor/sentry.svg";
  import { NextApiRequest } from "next";
  
  export class Sentry implements ObservabilityVendor {
    name = "sentry";
    displayName = "Sentry";
    type = VendorType.MANAGED;
    supportedSignals = [ObservabilitySignals.Traces];
    getLogo = (props: any) => {
      return <SentryLogo {...props} />;
    };
  
    getFields = (selectedSignals: any) => {
      return [
        {
          displayName: "DSN",
          id: "dsn",
          name: "dsn",
          type: "password",
        },
      ];
    };
  
    toObjects = (req: NextApiRequest) => {
      return {
        Secret: {
          DSN: Buffer.from(
            req.body.dsn
          ).toString("base64"),
        },
      };
    };
  
    mapDataToFields = (data: any) => {
      return {};
    };
  }
  