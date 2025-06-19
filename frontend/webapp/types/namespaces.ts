export interface NamespaceInstrumentInput {
  namespaces: {
    namespace: string;
    selected: boolean;
    currentStreamName: string;
  }[];
}
