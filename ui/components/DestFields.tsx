interface IDestField {
  displayName: string;
  id: string;
  name: string;
  type: string;
}

const DestinationFields: { [destname: string]: IDestField[] } = {
  grafana: [
    {
      displayName: "URL",
      id: "url",
      name: "url",
      type: "url",
    },
    {
      displayName: "User",
      id: "user",
      name: "user",
      type: "text",
    },
    {
      displayName: "API Key",
      id: "apikey",
      name: "apikey",
      type: "password",
    },
  ],
  honeycomb: [
    {
      displayName: "API Key",
      id: "apikey",
      name: "apikey",
      type: "password",
    },
  ],
  datadog: [
    {
      displayName: "Site",
      id: "site",
      name: "site",
      type: "text",
    },
    {
      displayName: "API Key",
      id: "apikey",
      name: "apikey",
      type: "password",
    },
  ],
};

export default DestinationFields;
