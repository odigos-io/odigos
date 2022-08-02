import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorObjects,
  VendorType,
} from "@/vendors/index";
import LokiLogo from "@/img/vendor/loki.svg";
import { NextApiRequest } from "next";

export class Loki implements ObservabilityVendor {
  name = "loki";
  displayName = "Loki";
  type = VendorType.HOSTED;
  supportedSignals = [ObservabilitySignals.Logs];

  getLogo = (props: any) => {
    return <LokiLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
      {
        displayName: "Loki URL",
        id: "loki_url",
        name: "loki_url",
        type: "url",
      },
    ];
  };

  toObjects = (req: NextApiRequest) => {
    return {
      Data: {
        LOKI_URL: req.body.loki_url,
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      loki_url: data.LOKI_URL || "",
    };
  };
}
