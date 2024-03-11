export interface ManagedSource {
  kind: string;
  name: string;
  namespace: string;
  languages: [
    {
      container_name: string;
      language: string;
    }
  ];
}
