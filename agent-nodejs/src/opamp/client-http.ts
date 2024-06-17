import { PartialMessage } from "@bufbuild/protobuf";
import { AgentDescription, AgentToServer, ServerToAgent } from "./generated/opamp_pb";
import { OpAMPClientHttpConfig, ResourceAttributeFromServer } from "./types";
import { keyValuePairsToOtelAttributes, otelAttributesToKeyValuePairs } from "./utils";
import { uuidv7 } from "uuidv7";
import axios, { AxiosInstance } from "axios";
import { DetectorSync, IResource, Resource, ResourceDetectionConfig } from "@opentelemetry/resources";
import { Attributes } from "@opentelemetry/api";

export class OpAMPClientHttp implements DetectorSync {
  private config: OpAMPClientHttpConfig;
  private instanceUid: Uint8Array;
  private nextSequenceNum: bigint = BigInt(0);
  private httpClient: AxiosInstance;

  // promise that we can resolve async later on, which the detect function can return
  private resourcePromiseResolver?: (resourceAttributes: Attributes) => void;

  constructor(config: OpAMPClientHttpConfig) {
    this.config = config;
    this.instanceUid = new TextEncoder().encode(uuidv7());
    this.httpClient = axios.create({
      baseURL: `http://${this.config.opAMPServerHost}`,
      headers: { 
        "Content-Type": " application/x-protobuf",
        "Authorization": `DeviceId ${config.instrumentationDeviceId}`,
       },
    });

    const timer = setInterval(() => {
      const heartbeatRes = this.sendHeartBeatToServer();
      console.log("Heartbeat response:", heartbeatRes);
    }, this.config.pollingIntervalMs || 30000);
    timer.unref(); // do not keep the process alive just for this timer
  }

  detect(): IResource {
    return new Resource({}, new Promise<Attributes>((resolve) => {
      this.resourcePromiseResolver = resolve;
    }));
  }

  async start() {
    try {
        const firstServerToAgent = await this.sendFirstAgentToServer();

        const resourceAttributes = JSON.parse(firstServerToAgent.remoteConfig?.config?.configMap['server-resolved-resource-attributes'].body?.toString() || "{}") as Array<ResourceAttributeFromServer>;
        if (this.resourcePromiseResolver) {
          console.log('Got resource attributes, resolving detector promise');
          this.resourcePromiseResolver(keyValuePairsToOtelAttributes(resourceAttributes));
        }

        console.log("Resource Attributes: ", resourceAttributes);

    } catch (error) {
        // TODO: handle
        console.log(error);
    }
  }

  private async sendHeartBeatToServer() {
    return await this.sendAgentToServerMessage({});
  }

  private async sendFirstAgentToServer() {
    return await this.sendAgentToServerMessage({
      // agent description is only sent in the first AgentToServer message
      agentDescription: new AgentDescription({
        identifyingAttributes: otelAttributesToKeyValuePairs(
          this.config.agentDescriptionIdentifyingAttributes
        ),
        nonIdentifyingAttributes: otelAttributesToKeyValuePairs(
          this.config.agentDescriptionNonIdentifyingAttributes
        ),
      }),
    });
  }

  private async sendAgentToServerMessage(
    message: PartialMessage<AgentToServer>
  ): Promise<ServerToAgent> {
    const completeMessageToSend = new AgentToServer({
      ...message,
      instanceUid: this.instanceUid,
      sequenceNum: this.nextSequenceNum++,
    });
    const msgBytes = completeMessageToSend.toBinary();
    try {
        const res = await this.httpClient.post("/v1/opamp", msgBytes, { responseType: "arraybuffer" });
        const agentToServer = ServerToAgent.fromBinary(res.data);
        return agentToServer;
    } catch (error) {
        // TODO: handle
        throw error;
    }
  }
}
