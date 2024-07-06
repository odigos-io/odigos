import { Instrumentation } from "@opentelemetry/instrumentation";
import { getNodeAutoInstrumentations } from "@opentelemetry/auto-instrumentations-node";
import { ProxyTracerProvider, TracerProvider, diag } from "@opentelemetry/api";
import { InstrumentationLibraryConfiguration } from "./opamp";
import { PackageStatus } from "./opamp/generated/opamp_pb";
import { PartialMessage } from "@bufbuild/protobuf";

type OdigosInstrumentation = {
  otelInstrumentation: Instrumentation;
};

export class InstrumentationLibraries {
  private instrumentations: Instrumentation[];
  private instrumentationLibraries: Map<string, OdigosInstrumentation>;

  private noopTracerProvider: TracerProvider;
  private tracerProvider: TracerProvider;

  private packageStatusCallback?: (status: PackageStatus[]) => void;

  private logger = diag.createComponentLogger({
    namespace: "@odigos/opentelemetry-node/instrumentation-libraries",
  });

  constructor() {
    this.instrumentations = getNodeAutoInstrumentations();

    // trick to get the noop tracer provider which is not exported from @openetelemetry/api
    this.noopTracerProvider = new ProxyTracerProvider().getDelegate();
    this.tracerProvider = this.noopTracerProvider; // starts as noop, and overridden later on

    this.instrumentationLibraries = new Map(
      this.instrumentations.map((otelInstrumentation) => {
        // start all instrumentations with a noop tracer provider
        otelInstrumentation.setTracerProvider(this.noopTracerProvider);

        const { instrumentationName } = otelInstrumentation;
        const odigosInstrumentation = {
          otelInstrumentation,
        };

        return [instrumentationName, odigosInstrumentation];
      })
    );
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

  public applyNewConfig(configs: InstrumentationLibraryConfiguration[]) {
    for (const instrumentationLibraryConfig of configs) {
      const odigosInstrumentation = this.instrumentationLibraries.get(
        instrumentationLibraryConfig.name
      );
      if (!odigosInstrumentation) {
        this.logger.error('remote config instrumentation name not found:', instrumentationLibraryConfig.name);
        continue;
      }

      this.logger.info('applying new instrumentation library config:', {instrumentationName: instrumentationLibraryConfig.name, enabled: instrumentationLibraryConfig.enabled} );
      if (instrumentationLibraryConfig.enabled) {
        odigosInstrumentation.otelInstrumentation.setTracerProvider(
          this.tracerProvider
        );
      } else {
        odigosInstrumentation.otelInstrumentation.setTracerProvider(
          this.noopTracerProvider
        );
      }
    }
  }
}
