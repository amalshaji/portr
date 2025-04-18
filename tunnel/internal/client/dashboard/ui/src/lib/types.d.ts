export type Tunnel = {
  Subdomain: string;
  Localport: number;
};

export type Request = {
  ID: string;
  Subdomain: string;
  Host: string;
  Localport: number;
  Url: string;
  Method: string;
  Headers: Record<string, string>;
  Body: string;
  ResponseStatusCode: number;
  ResponseHeaders: Record<string, string>;
  ResponseBody: string;
  IsReplayed: boolean;
  ParentID: string;
  LoggedAt: string;
};
