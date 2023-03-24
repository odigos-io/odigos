import {
    ObservabilityVendor,
    ObservabilitySignals,
    VendorType,
} from "@/vendors/index";
import MiddlewareLogo from "@/img/vendor/middleware.svg";
import { NextApiRequest } from "next";

export class Middleware implements ObservabilityVendor {
    name = "middleware";
    displayName = "Middleware";
    type = VendorType.MANAGED;
    supportedSignals = [
        ObservabilitySignals.Traces,
        ObservabilitySignals.Metrics,
        ObservabilitySignals.Logs,
    ];
  
    getLogo = (props: any) => {
      return <MiddlewareLogo {...props} />;
    };

    getFields = (selectedSignals: any) => {
        return [
          {
            displayName: "Endpoint",
            id: "target",
            name: "target",
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
            MW_TARGET: req.body.target,
          },
          Secret: {
            MW_API_KEY: Buffer.from(req.body.apikey).toString("base64"),
          },
        };
      };

      mapDataToFields = (data: any) => {
        return {
          target: data.MW_TARGET,
        };
      };
}
  
  