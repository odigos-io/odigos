import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorType,
} from "@/vendors/index";
import ChronosphereLogo from "@/img/vendor/chronosphere.svg";
import { NextApiRequest } from "next";

export class Chronosphere implements ObservabilityVendor {
  name = "chronosphere";
  displayName = "Chronosphere";
  type = VendorType.MANAGED;
  supportedSignals = [ObservabilitySignals.Traces, ObservabilitySignals.Metrics];
  getLogo = (props: any) => {
    return <ChronosphereLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
      {
        displayName: "Chronosphere Collector Name",
        id: "chronosphere_collector",
        name: "chronosphere_collector",
        type: "text",
      },
    ];
  };

  toObjects = (req: NextApiRequest) => {
    return {
      Data: {
        CHRONOSPHERE_COLLECTOR: req.body.chronosphere_collector,
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      chronosphere_collector: data.CHRONOSPHERE_COLLECTOR || "",
    };
  };
}
