import { Instrumentation } from "@opentelemetry/instrumentation";
import { getNodeAutoInstrumentations } from "@opentelemetry/auto-instrumentations-node";
import { ProxyTracerProvider, TracerProvider, diag } from "@opentelemetry/api";
import { InstrumentationLibraryConfiguration } from "./opamp";
import { PackageStatus } from "./opamp/generated/opamp_pb";
import { PartialMessage } from "@bufbuild/protobuf";

type OdigosInstrumentation = {
  otelInstrumentation: Instrumentation;
};

const calculateLibraryEnabled = (remoteConfigDisabledOption: boolean | undefined, librariesDefaultEnable: boolean): boolean => {
  if (remoteConfigDisabledOption !== undefined) {
    return !remoteConfigDisabledOption;
  } else {
    // if there is no remote config to enable/disable this library, use the default
    return librariesDefaultEnable;
  }
}

export class InstrumentationLibraries {
  private instrumentations: Instrumentation[];
  private instrumentationLibraries: OdigosInstrumentation[];

  private noopTracerProvider: TracerProvider;
  private tracerProvider: TracerProvider;

  private logger = diag.createComponentLogger({
    namespace: "@odigos/opentelemetry-node/instrumentation-libraries",
  });

  constructor() {
    this.instrumentations = getNodeAutoInstrumentations();

    // trick to get the noop tracer provider which is not exported from @openetelemetry/api
    this.noopTracerProvider = new ProxyTracerProvider().getDelegate();
    this.tracerProvider = this.noopTracerProvider; // starts as noop, and overridden later on

    this.instrumentationLibraries =
      this.instrumentations.map((otelInstrumentation) => {
        // start all instrumentations with a noop tracer provider
        otelInstrumentation.setTracerProvider(this.noopTracerProvider);

        const odigosInstrumentation = {
          otelInstrumentation,
        };

        return odigosInstrumentation;
      });
  }

  public getPackageStatuses(): PartialMessage<PackageStatus>[] {
    return this.instrumentations.map((instrumentation) => {
      return {
        name: instrumentation.instrumentationName,
        agentHasVersion: instrumentation.instrumentationVersion,
      };
    });
  }

  public setTracerProvider(tracerProvider: TracerProvider) {
    this.tracerProvider = tracerProvider;
  }

  public applyNewConfig(configs: InstrumentationLibraryConfiguration[], librariesDefaultEnable: boolean) {

    // make the configs into a map by library name so it's quicker to find the right one
    const configsMap = new Map<string, InstrumentationLibraryConfiguration>(
      configs.map((config) => [config.name, config])
    );

    for(const odigosInstrumentation of this.instrumentationLibraries) {
      const config = configsMap.get(odigosInstrumentation.otelInstrumentation.instrumentationName);
      const enabled = calculateLibraryEnabled(config?.traces?.disabled, librariesDefaultEnable);
      const tracerProvider = enabled ? this.tracerProvider : this.noopTracerProvider;
      odigosInstrumentation.otelInstrumentation.setTracerProvider(tracerProvider);
    }
  }
}
