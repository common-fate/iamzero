export interface Event {
  time: string;
  data: AWSData;
  // identity: AWSIdentity;
}

export interface AWSData {
  service: string;
  region: string;
  operation: string;
  /**
   * Parameters may be a string, or an object themselves.
   *  An example is dynamodb keys which are in the form "key": {"id": "1"}
   */
  parameters: Record<string, string | Record<string, unknown>>;
  exceptionMessage: string;
  exceptionCode: string;
}

export interface AWSIdentity {
  user: string;
  role: string;
  account: string;
}

export interface Recommendation {
  Description?: RecommendationDescription[];
  AWSPolicy?: AWSIAMPolicy;
  Comment: string;
  ID: string;
}

export interface RecommendationDescription {
  AppliedTo: string;
  Type: string;
  Policy: Record<string, any>;
}

export interface AWSIAMPolicy {
  Version: "2012-10-17";
  Statement: AWSIAMStatement[];
}

export interface AWSIAMStatement {
  Sid: string;
  Effect: "Allow";
  Action: string | string[];
  Resource: string[];
}

export type AlertStatus = "active" | "fixed" | "applying" | "ignored";

/** A resource such as an AWS S3 bucket */
export interface Resource {
  id: string;
  name: string;
}

/** An alert that iam-zero has generated recommendations for */
export interface ActionWithRecommendations {
  id: string;
  findingId: string;
  event: Event;
  time: Date;
  status: AlertStatus;
  recommendations: Recommendation[];
  resources: Resource[];
  hasRecommendations: true;
  enabled: boolean;
  selectedAdvisoryId: string;
}

/** An alert that we do not yet handle and haven't generated recommendations for */
export interface UnhandledAction {
  id: string;
  event: Event;
  findingId: string;
  time: Date;
  status: AlertStatus;
  recommendations: null;
  resources: Resource[];
  hasRecommendations: false;
  enabled: boolean;
  selectedAdvisoryId: string;
}

export type Action = ActionWithRecommendations | UnhandledAction;

export interface Token {
  id: string;
  name: string;
}

/**
 * A least-privilege Finding generated by IAM Zero
 */
export interface Finding {
  id: string;
  identity: AWSIdentity;
  lastUpdated: Date;
  eventCount: number;
  document: AWSIAMPolicy;
  status: PolicyStatus;
}

export type PolicyStatus = "active" | "resolved";
