import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorType,
} from "@/vendors/index";
import GoogleCloudLogo from "@/img/vendor/gcp.svg";
import { NextApiRequest } from "next";

export class GoogleCloud implements ObservabilityVendor {
  name = "googlecloud";
  displayName = "Google Cloud";
  type = VendorType.MANAGED;
  supportedSignals = [ObservabilitySignals.Traces, ObservabilitySignals.Logs];
  getLogo = (props: any) => {
    return <GoogleCloudLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    return [];
  };

  toObjects = (req: NextApiRequest) => {
    return {};
  };

  mapDataToFields = (data: any) => {
    return {};
  };
}
