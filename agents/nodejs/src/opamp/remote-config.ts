import { Resource } from "@opentelemetry/resources";
import { AgentRemoteConfig } from "./generated/opamp_pb";
import { InstrumentationLibraryConfiguration, RemoteConfig } from "./types";
import { keyValuePairsToOtelAttributes } from "./utils";
import { SEMRESATTRS_SERVICE_INSTANCE_ID } from "@opentelemetry/semantic-conventions";
import { OpAMPSdkConfiguration } from "./opamp-types";

export const extractRemoteConfigFromResponse = (
  agentRemoteConfig: AgentRemoteConfig,
  instanceUid: string
): RemoteConfig => {
  const instrumentationLibrariesConfigSection =
    agentRemoteConfig.config?.configMap["InstrumentationLibraries"];
  if (
    !instrumentationLibrariesConfigSection ||
    !instrumentationLibrariesConfigSection.body
  ) {
    throw new Error("missing instrumentation libraries remote config");
  }
  const instrumentationLibrariesConfigBody =
    instrumentationLibrariesConfigSection.body.toString();

  let instrumentationLibrariesConfig: InstrumentationLibraryConfiguration[];
  try {
    instrumentationLibrariesConfig = JSON.parse(
      instrumentationLibrariesConfigBody
    ) as InstrumentationLibraryConfiguration[];
  } catch (error) {
    throw new Error("error parsing instrumentation libraries remote config");
  }

  const sdkConfigSection = agentRemoteConfig.config?.configMap["SDK"];
  if (!sdkConfigSection || !sdkConfigSection.body) {
    throw new Error("missing SDK remote config");
  }
  const sdkConfigBody = sdkConfigSection.body.toString();

  let sdkConfig: OpAMPSdkConfiguration;
  try {
    sdkConfig = JSON.parse(sdkConfigBody) as OpAMPSdkConfiguration;
  } catch (error) {
    throw new Error("error parsing SDK remote config");
  }

  const remoteResource = new Resource(
    keyValuePairsToOtelAttributes([
      ...sdkConfig.remoteResourceAttributes,
      {
        key: SEMRESATTRS_SERVICE_INSTANCE_ID,
        value: instanceUid,
      },
    ])
  );

  return {
    sdk: {
      remoteResource,
      traceSignal: sdkConfig.traceSignal,
    },
    instrumentationLibraries: instrumentationLibrariesConfig,
  };
};
