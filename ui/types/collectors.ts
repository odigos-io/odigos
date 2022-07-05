export interface Collector {
  name: string;
  ready: boolean;
}

export interface ICollectorsResponse {
  collectors: Collector[];
}
