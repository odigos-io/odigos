export interface TokenPayload {
  token: string;
  aud: string;
  iat: number;
  exp: number;
}

export interface GetApiTokens {
  getApiTokens: TokenPayload[];
}
