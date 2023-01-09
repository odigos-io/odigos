import {
    ObservabilityVendor,
    ObservabilitySignals,
    VendorType,
  } from "@/vendors/index";
  import OpsVerseLogo from "@/img/vendor/opsverse.svg";
  import { NextApiRequest } from "next";

  export class OpsVerse implements ObservabilityVendor {
    name = "opsverse";
    displayName = "OpsVerse";
    type = VendorType.MANAGED;
    supportedSignals = [
      ObservabilitySignals.Traces,
      ObservabilitySignals.Metrics,
      ObservabilitySignals.Logs,
    ];

    getLogo = (props: any) => {
      return <OpsVerseLogo {...props} />;
    };

    getFields = (selectedSignals: any) => {
      let fields = [
        {
          displayName: "User",
          id: "user",
          name: "user",
          type: "text",
        },
        {
          displayName: "Password",
          id: "password",
          name: "password",
          type: "password",
        },
      ];

      if (selectedSignals[ObservabilitySignals.Logs]) {
        fields.push({
          displayName: "Logs Endpoint",
          id: "logsUrl",
          name: "logsUrl",
          type: "url",
        })
      }
      if (selectedSignals[ObservabilitySignals.Traces]) {
        fields.push({
          displayName: "Traces (OTLP) Endpoint",
          id: "tracesUrl",
          name: "tracesUrl",
          type: "url",
        })
      }
      if (selectedSignals[ObservabilitySignals.Metrics]) {
        fields.push({
          displayName: "Metrics Endpoint",
          id: "metricsUrl",
          name: "metricsUrl",
          type: "url",
        })
      }

      return fields;
    };

    toObjects = (req: NextApiRequest) => {
      // Exporters expect token to be base64 encoded, therefore we encode twice.
      const authString = Buffer.from(
        `${req.body.user}:${req.body.password}`
      ).toString("base64");

      return {
        Data: {
          OPSVERSE_LOGS_URL: req.body.logsUrl,
          OPSVERSE_METRICS_URL: req.body.metricsUrl,
          OPSVERSE_TRACES_URL: req.body.tracesUrl,
          OPSVERSE_USERNAME: req.body.user,
        },
        Secret: {
          OPSVERSE_AUTH_TOKEN: Buffer.from(authString).toString("base64"),
          OPSVERSE_PASSWORD: Buffer.from(req.body.password).toString("base64"),
        },
      };
    };

    mapDataToFields = (data: any) => {
      return {
        logsUrl: data.OPSVERSE_LOGS_URL || "",
        metricsUrl: data.OPSVERSE_METRICS_URL || "",
        tracesUrl: data.OPSVERSE_TRACES_URL || "",
        user: data.OPSVERSE_USERNAME || "",
      };
    };
  }