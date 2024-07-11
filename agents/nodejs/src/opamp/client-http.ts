import { PartialMessage } from "@bufbuild/protobuf";
import {
  AgentDescription,
  AgentToServer,
  RemoteConfigStatus,
  RemoteConfigStatuses,
  ServerToAgent,
  ServerToAgentFlags,
} from "./generated/opamp_pb";
import {
  InstrumentationLibraryConfiguration,
  OpAMPClientHttpConfig,
  ResourceAttributeFromServer,
} from "./types";
import {
  keyValuePairsToOtelAttributes,
  otelAttributesToKeyValuePairs,
} from "./utils";
import { uuidv7 } from "uuidv7";
import axios, { AxiosInstance } from "axios";
import { Resource } from "@opentelemetry/resources";
import { context, diag } from "@opentelemetry/api";
import {
  SEMRESATTRS_SERVICE_INSTANCE_ID,
  SEMRESATTRS_SERVICE_NAME,
} from "@opentelemetry/semantic-conventions";
import { suppressTracing } from "@opentelemetry/core";
import { PackageStatuses } from "./generated/opamp_pb";

export class OpAMPClientHttp {
  private config: OpAMPClientHttpConfig;
  private OpAMPInstanceUidString: string;
  private OpAMPInstanceUid: Uint8Array;
  private nextSequenceNum: bigint = BigInt(0);
  private httpClient: AxiosInstance;
  private logger = diag.createComponentLogger({
    namespace: "@odigos/opentelemetry-node/opamp",
  });
  private remoteConfigStatus: RemoteConfigStatus | undefined;

  constructor(config: OpAMPClientHttpConfig) {
    this.config = config;
    this.OpAMPInstanceUidString = uuidv7();
    this.OpAMPInstanceUid = new TextEncoder().encode(
      this.OpAMPInstanceUidString
    );
    this.httpClient = axios.create({
      baseURL: `http://${this.config.opAMPServerHost}`,
      headers: {
        "Content-Type": " application/x-protobuf",
        "X-Odigos-DeviceId": config.instrumentationDeviceId,
      },
      timeout: 5000,
    });
    this.httpClient.interceptors.response.use(
      (response) => response,
      (error) => {
        const relevantErrorInfo = {
          message: error.message,
          code: error.code,
          status: error.response?.status,
          statusText: error.response?.statusText,
          data: error.response?.data,
          url: error.config?.url,
          method: error.config?.method,
          headers: error.config?.headers,
        };
        return Promise.reject(relevantErrorInfo);
      }
    );
  }

  async start() {
    await this.sendFirstMessageWithRetry();
    const timer = setInterval(async () => {
      let heartbeatRes = await this.sendHeartBeatToServer();
      if (!heartbeatRes) {
        return;
      }

      // flags is bitmap, use bitwise OR to check if the flag is set
      if (
        Number(heartbeatRes.flags) |
        Number(ServerToAgentFlags.ServerToAgentFlags_ReportFullState)
      ) {
        this.logger.info("Opamp server requested full state report");
        heartbeatRes = await this.sendFullState();
      }
    }, this.config.pollingIntervalMs || 30000);
    timer.unref(); // do not keep the process alive just for this timer
  }

  async shutdown() {
    this.logger.info("Sending AgentDisconnect message to OpAMP server");
    return await this.sendAgentToServerMessage({
      agentDisconnect: {},
    });
  }

  // the first opamp message is special, as we need to get the remote resource attributes.
  // this function will attempt to send the first message, and will retry after some interval if it fails.
  // if no remote resource attributes are received after some grace period, we will continue without them.
  private async sendFirstMessageWithRetry() {
    let remainingRetries = 5;
    const retryIntervalMs = 2000;

    for (let i = 0; i < remainingRetries; i++) {
      try {
        const firstServerToAgent = await this.sendFullState();
        this.handleFirstMessageResponse(firstServerToAgent);
        this.handleRemoteConfigInResponse(firstServerToAgent);
        return;
      } catch (error) {
        this.logger.warn(
          `Error sending first message to OpAMP server, retrying in ${retryIntervalMs}ms`,
          error
        );
        await new Promise((resolve) => {
          const timer = setTimeout(resolve, retryIntervalMs);
          timer.unref(); // do not keep the process alive just for this timer
        });
      }
    }

    // if we got here, it means we run out of retries and did not return from the loop
    // at this point we have no remote resource attributes, so we set the service name to the instrumentation device id
    // which is the best we can do without the remote resource attributes
    this.logger.error(
      `Failed to get remote resource attributes from OpAMP server after retries, continuing without them`
    );
    this.config.onRemoteResource?.(new Resource({
      [SEMRESATTRS_SERVICE_NAME]: this.config.instrumentationDeviceId,
    }));
  }

  private handleFirstMessageResponse(serverToAgentMessage: ServerToAgent) {
    const sdkConfig =
      serverToAgentMessage.remoteConfig?.config?.configMap["SDK"];
    if (!sdkConfig || !sdkConfig.body) {
      throw new Error(
        "No SDK config received on first OpAMP message to the server"
      );
    }

    const resourceAttributes = JSON.parse(sdkConfig.body.toString()) as {
      remoteResourceAttributes: ResourceAttributeFromServer[];
    };

    this.logger.info(
      "Got remote resource attributes",
      resourceAttributes.remoteResourceAttributes
    );

    const remoteResource = new Resource(
      keyValuePairsToOtelAttributes([
        ...resourceAttributes.remoteResourceAttributes,
        {
          key: SEMRESATTRS_SERVICE_INSTANCE_ID,
          value: this.OpAMPInstanceUidString,
        },
      ])
    );
    this.config.onRemoteResource?.(remoteResource);
  }

  private handleRemoteConfigInResponse(serverToAgentMessage: ServerToAgent) {

    const remoteConfig = serverToAgentMessage.remoteConfig;
    if (!remoteConfig) {
      return;
    }

    this.logger.info("Got new remote config from OpAMP server");

    const instrumentationLibrariesConfig =
      remoteConfig.config?.configMap["InstrumentationLibraries"];
    if (
      !instrumentationLibrariesConfig ||
      !instrumentationLibrariesConfig.body
    ) {
      this.remoteConfigStatus = new RemoteConfigStatus({
        lastRemoteConfigHash: remoteConfig.configHash,
        status: RemoteConfigStatuses.RemoteConfigStatuses_FAILED,
        errorMessage: 'missing instrumentation libraries remote config',
      });
      return;
    }

    const instrumentationLibrariesConfigBody =
      instrumentationLibrariesConfig.body.toString();
    try {
      const configs = JSON.parse(
        instrumentationLibrariesConfigBody
      ) as InstrumentationLibraryConfiguration[];
      this.config.onNewInstrumentationLibrariesConfiguration?.(configs);
    } catch (error) {
      this.remoteConfigStatus = new RemoteConfigStatus({
        lastRemoteConfigHash: remoteConfig.configHash,
        status: RemoteConfigStatuses.RemoteConfigStatuses_FAILED,
        errorMessage: 'error parsing instrumentation libraries remote config',
      });

      this.logger.warn("Error handling instrumentation libraries remote config", error);
    }

    this.remoteConfigStatus = new RemoteConfigStatus({
      lastRemoteConfigHash: remoteConfig.configHash,
      status: RemoteConfigStatuses.RemoteConfigStatuses_APPLIED,
    });
  }

  private async sendHeartBeatToServer() {
    try {
      return await this.sendAgentToServerMessage({});
    } catch (error) {
      this.logger.warn("Error sending heartbeat to OpAMP server", error);
    }
  }

  private async sendFullState() {
    return await this.sendAgentToServerMessage({
      agentDescription: new AgentDescription({
        identifyingAttributes: otelAttributesToKeyValuePairs({
          [SEMRESATTRS_SERVICE_INSTANCE_ID]: this.OpAMPInstanceUidString, // always send the instance id
          ...this.config.agentDescriptionIdentifyingAttributes,
        }),
        nonIdentifyingAttributes: otelAttributesToKeyValuePairs(
          this.config.agentDescriptionNonIdentifyingAttributes
        ),
      }),
      packageStatuses: new PackageStatuses({
        packages: Object.fromEntries(this.config.initialPackageStatues.map((pkg) => [pkg.name, pkg])),
      }),
    });
  }

  private async sendAgentToServerMessage(
    message: PartialMessage<AgentToServer>
  ): Promise<ServerToAgent> {
    const completeMessageToSend = new AgentToServer({
      ...message,
      instanceUid: this.OpAMPInstanceUid,
      sequenceNum: this.nextSequenceNum++,
      remoteConfigStatus: this.remoteConfigStatus,
    });
    const msgBytes = completeMessageToSend.toBinary();
    try {
      // do not create traces for the opamp http requests
      return context.with(suppressTracing(context.active()), async () => {
        const res = await this.httpClient.post("/v1/opamp", msgBytes, {
          responseType: "arraybuffer",
        });
        const serverToAgent = ServerToAgent.fromBinary(res.data);
        this.handleRemoteConfigInResponse(serverToAgent);
        return serverToAgent;
      });
    } catch (error) {
      // TODO: handle
      throw error;
    }
  }
}
