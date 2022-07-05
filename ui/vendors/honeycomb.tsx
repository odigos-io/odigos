import { ObservabilitySignals, ObservabilityVendor } from "@/vendors/index";
import HoneycombLogo from "@/img/vendor/honeycomb.svg";

export class Honeycomb implements ObservabilityVendor {
  name = "honeycomb";
  displayName = "Honeycomb";
  supportedSignals = [
    ObservabilitySignals.Metrics,
    ObservabilitySignals.Traces,
  ];

  getLogo = (props: any) => {
    return <HoneycombLogo {...props} />;
  };

  getFields = () => {
    return [
      {
        displayName: "API Key",
        id: "apikey",
        name: "apikey",
        type: "password",
      },
    ];
  };
}
