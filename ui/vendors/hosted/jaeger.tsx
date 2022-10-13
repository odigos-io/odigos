import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorObjects,
  VendorType,
  IDestField,
} from "@/vendors/index";
import JaegerLogo from "@/img/vendor/jaeger.svg";
import { NextApiRequest } from "next";

export class Jaeger implements ObservabilityVendor {
  name = "jaeger";
  displayName = "Jaeger";
  type = VendorType.HOSTED;
  supportedSignals = [ObservabilitySignals.Traces];
  getLogo = (props: any) => {
    return <JaegerLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
      {
        displayName: "Jaeger URL",
        id: "jaeger_url",
        name: "jaeger_url",
        type: "url",
      },
    ];
  };

  toObjects = (req: NextApiRequest) => {
    return {
      Data: {
        JAEGER_URL: req.body.jaeger_url,
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      jaeger_url: data.JAEGER_URL || "",
    };
  };
}
