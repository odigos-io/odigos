import {
  ObservabilityVendor,
  ObservabilitySignals,
  VendorObjects,
} from "@/vendors/index";
import GrafanaLogo from "@/img/vendor/grafana.svg";
import { NextApiRequest } from "next";

export class Grafana implements ObservabilityVendor {
  name = "grafana";
  displayName = "Grafana Cloud";
  supportedSignals = [
    ObservabilitySignals.Traces,
    ObservabilitySignals.Metrics,
    ObservabilitySignals.Logs,
  ];

  getLogo = (props: any) => {
    return <GrafanaLogo {...props} />;
  };

  getFields = (selectedSignals: any) => {
    let fields = [
      {
        displayName: "API Key",
        id: "apikey",
        name: "apikey",
        type: "password",
      },
    ];

    if (selectedSignals[ObservabilitySignals.Traces]) {
      fields.push(
        {
          displayName: "Tempo URL",
          id: "tempo_url",
          name: "tempo_url",
          type: "url",
        },
        {
          displayName: "Tempo User",
          id: "tempo_user",
          name: "tempo_user",
          type: "text",
        }
      );
    }

    if (selectedSignals[ObservabilitySignals.Metrics]) {
      fields.push(
        {
          displayName: "Remote Write URL",
          id: "remotewrite_url",
          name: "remotewrite_url",
          type: "url",
        },
        {
          displayName: "Prometheus User",
          id: "metrics_user",
          name: "metrics_user",
          type: "text",
        }
      );
    }

    if (selectedSignals[ObservabilitySignals.Logs]) {
      fields.push(
        {
          displayName: "Loki URL",
          id: "loki_url",
          name: "loki_url",
          type: "url",
        },
        {
          displayName: "Loki User",
          id: "loki_user",
          name: "loki_user",
          type: "text",
        }
      );
    }

    return fields;
  };

  toObjects = (req: NextApiRequest) => {
    // Tempo exporter expect token to be bas64 encoded, therefore we encode twice.
    const authString = Buffer.from(
      `${req.body.tempo_user}:${req.body.apikey}`
    ).toString("base64");

    return {
      Data: {
        GRAFANA_TEMPO_URL: req.body.tempo_url,
        GRAFANA_REMOTEWRITE_URL: req.body.remotewrite_url,
        GRAFANA_METRICS_USER: req.body.metrics_user,
        GRAFANA_LOKI_USER: req.body.loki_user,
        GRAFANA_LOKI_URL: req.body.loki_url,
      },
      Secret: {
        GRAFANA_API_KEY: Buffer.from(req.body.apikey).toString("base64"),
        GRAFANA_TEMPO_AUTH_TOKEN: Buffer.from(authString).toString("base64"),
      },
    };
  };

  mapDataToFields = (data: any) => {
    return {
      tempo_url: data.GRAFANA_TEMPO_URL,
      remotewrite_url: data.GRAFANA_REMOTEWRITE_URL,
      loki_url: data.GRAFANA_LOKI_URL,
    };
  };
}
