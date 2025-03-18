type Req = {
  alias?: string;
  body: {
    operationName?: string;
  };
};

// Utility to match GraphQL mutation based on the operation name
export const hasOperationName = (req: Req, operationName: string) => {
  const { body } = req;
  return Object.prototype.hasOwnProperty.call(body, 'operationName') && body.operationName === operationName;
};

// Alias query if operationName matches
export const aliasQuery = (req: Req, operationName: string) => {
  if (hasOperationName(req, operationName)) {
    req.alias = `gql${operationName}Query`;
  }
};

// Alias mutation if operationName matches
export const aliasMutation = (req: Req, operationName: string) => {
  if (hasOperationName(req, operationName)) {
    req.alias = `gql${operationName}Mutation`;
  }
};
