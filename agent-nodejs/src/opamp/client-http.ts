import { PartialMessage } from "@bufbuild/protobuf";
import { AgentDescription, AgentToServer, ServerToAgent } from "./generated/opamp_pb";
import { OpAMPClientHttpConfig } from "./types";
import { otelAttributesToKeyValuePairs } from "./utils";
import { uuidv7 } from "uuidv7";
import axios, { AxiosInstance } from "axios";

export class OpAMPClientHttp {
  private config: OpAMPClientHttpConfig;
  private instanceUid: Uint8Array;
  private nextSequenceNum: bigint = BigInt(0);
  private httpClient: AxiosInstance;

  constructor(config: OpAMPClientHttpConfig) {
    this.config = config;
    this.instanceUid = new TextEncoder().encode(uuidv7());
    this.httpClient = axios.create({
      baseURL: `http://${this.config.opAMPServerHost}`,
      headers: { "Content-Type": " application/x-protobuf" },
    });
  }

  async start() {
    try {
        const firstServerToAgent = await this.sendFirstAgentToServer();
    } catch (error) {
        // TODO: handle
        console.log(error);
    }
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
        console.log(agentToServer);
        return agentToServer;
    } catch (error) {
        // TODO: handle
        throw error;
    }
  }
}
