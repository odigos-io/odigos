import { Attributes } from "@opentelemetry/api";

export interface OpAMPClientHttpConfig {
    opAMPServerHost: string; // the host + (optional) port of the OpAMP server to connect over http://
    pollingInterval?: number;

    agentDescriptionIdentifyingAttributes?: Attributes;
    agentDescriptionNonIdentifyingAttributes?: Attributes;
}
