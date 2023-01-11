import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorType,
} from "@/vendors/index";
import LightstepLogo from "@/img/vendor/lightstep.svg";
import { NextApiRequest } from "next";

export class Lightstep implements ObservabilityVendor {
  name = "lightstep";
  displayName = "Lightstep";
  type = VendorType.MANAGED;
  supportedSignals = [ObservabilitySignals.Traces];
  getLogo = (props: any) => {
    return <LightstepLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
      {
        displayName: "Access Token",
        id: "lightstep_access_token",
        name: "lightstep_access_token",
        type: "password",
      },
    ];
  };

  toObjects = (req: NextApiRequest) => {
    return {
      Secret: {
        LIGHTSTEP_ACCESS_TOKEN: Buffer.from(
          req.body.lightstep_access_token
        ).toString("base64"),
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {};
  };
}
