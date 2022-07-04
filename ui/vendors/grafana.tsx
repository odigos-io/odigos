import { ObservabilityVendor, ObservabilitySignals } from "@/vendors/index";
import GrafanaLogo from "@/img/vendor/grafana.svg";

export class Grafana implements ObservabilityVendor {
  name = "grafana";
  displayName = "Grafana Cloud";
  supportedSignals = [
    ObservabilitySignals.Logs,
    ObservabilitySignals.Metrics,
    ObservabilitySignals.Traces,
  ];

  getLogo = (props: any) => {
    return <GrafanaLogo {...props} />;
  };

  getFields = () => {
    return [
      {
        displayName: "URL",
        id: "url",
        name: "url",
        type: "url",
      },
      {
        displayName: "User",
        id: "user",
        name: "user",
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
