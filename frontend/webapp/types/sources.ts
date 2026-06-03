export interface SourceInstrumentInput {
  sources: {
    namespace: string;
    name: string;
    kind: string;
    selected: boolean;
    currentStreamName: string;
  }[];
}
