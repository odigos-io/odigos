import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorType,
} from "@/vendors/index";
import SplunkLogo from "@/img/vendor/splunk.svg";
import { NextApiRequest } from "next";

export class Splunk implements ObservabilityVendor {
  name = "splunk";
  displayName = "Splunk";
  type = VendorType.MANAGED;
  supportedSignals = [ObservabilitySignals.Traces];
  getLogo = (props: any) => {
    return <SplunkLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [
      {
        displayName: "Access Token",
        id: "splunk_access_token",
        name: "splunk_access_token",
        type: "password",
      },
      {
        displayName: "Realm",
        id: "splunk_realm",
        name: "splunk_realm",
        type: "text",
      },
    ];
  };

  toObjects = (req: NextApiRequest) => {
    return {
      Data: {
        SPLUNK_REALM: req.body.splunk_realm,
      },
      Secret: {
        SPLUNK_ACCESS_TOKEN: Buffer.from(req.body.splunk_access_token).toString(
          "base64"
        ),
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      splunk_realm: data.SPLUNK_REALM || "",
    };
  };
}
