import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorObjects,
  VendorType,
} from "@/vendors/index";
import TempoLogo from "@/img/vendor/tempo.svg";
import { NextApiRequest } from "next";

export class Tempo implements ObservabilityVendor {
  name = "tempo";
  displayName = "Tempo";
  type = VendorType.HOSTED;
  supportedSignals = [ObservabilitySignals.Traces];

  getLogo = (props: any) => {
    return <TempoLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
      {
        displayName: "Tempo URL",
        id: "tempo_url",
        name: "tempo_url",
        type: "url",
      },
    ];
  };

  toObjects = (req: NextApiRequest) => {
    return {
      Data: {
        TEMPO_URL: req.body.tempo_url,
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      tempo_url: data.TEMPO_URL || "",
    };
  };
}
