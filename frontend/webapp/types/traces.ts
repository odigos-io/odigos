interface TraceTag {
  key: string;
  type: string;
  value: string;
}

interface TraceReference {
  refType: string;
  traceID: string;
  spanID: string;
}

interface TraceLog {
  timestamp: number;
  fields: TraceTag[];
}

interface TraceSpan {
  traceID: string;
  spanID: string;
  operationName: string;
  references: TraceReference[];
  startTime: number;
  duration: number;
  tags: TraceTag[];
  logs: TraceLog[];
  processID: string;
  warnings: string;
}

interface TraceProcess {
  serviceName: string;
  tags: TraceTag[];
}

export interface Trace {
  traceID: string;
  spans: TraceSpan[];
  processes: TraceProcess[];
  warnings: string;
}
