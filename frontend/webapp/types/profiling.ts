/** Mirrors backend `flamegraph.FlamebearerProfile` (Pyroscope JSON shape). */
export interface FlamebearerProfile {
  version: number;
  flamebearer: {
    names: string[];
    levels: number[][];
    numTicks: number;
    maxSelf: number;
  };
  metadata?: {
    format: string;
    spyName: string;
    sampleRate: number;
    units: string;
    name: string;
    symbolsHint?: string;
  };
  timeline?: {
    startTime: number;
    samples: number[];
    durationDelta: number;
    watermarks?: number[] | null;
  } | null;
  groups?: unknown;
  heatmap?: unknown;
}

