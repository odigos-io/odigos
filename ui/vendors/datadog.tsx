import { ObservabilityVendor, ObservabilitySignals } from "@/vendors/index";
import DatadogLogo from "@/img/vendor/datadog.svg";

export class Datadog implements ObservabilityVendor {
  name = "datadog";
  displayName = "Datadog";
  supportedSignals = [
    ObservabilitySignals.Logs,
    ObservabilitySignals.Metrics,
    ObservabilitySignals.Traces,
  ];

  getLogo = (props: any) => {
    return <DatadogLogo {...props} />;
  };

  getFields = () => {
    return [
      {
        displayName: "Site",
        id: "site",
        name: "site",
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
}
