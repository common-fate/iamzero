import useSWR from "swr";
import { Action, Policy, Token } from "./api-types";

/**
 * Adds the x-iamzero-token header to auth requests.
 * You can provide a token as an argument, otherwise it fetches the token from
 * localStorage.
 * Only adds the header if a token is found.
 */
export async function fetchWithAuth<T>(
  path: string,
  init?: RequestInit | undefined,
  token?: string
) {
  const authToken = token ?? localStorage.getItem("iamzeroToken");
  const headers = authToken
    ? { ...init?.headers, "x-iamzero-token": authToken }
    : init?.headers;

  const r = await fetch(path, {
    ...init,
    headers: {
      ...headers,
      "Accept": "application/json",
      "Content-Type": "application/json",
    },
  });
  return r.json() as Promise<T>;
}

export interface GetTokensResponse {
  tokens: Token[];
}

export const useTokens = () => useSWR<GetTokensResponse>("/api/v1/tokens");

export const deleteToken = (tokenId: string) =>
  fetchWithAuth(`/api/v1/tokens/${tokenId}`, {
    method: "DELETE",
  });

export const createToken = (name: string) =>
  fetchWithAuth(`/api/v1/tokens`, {
    method: "POST",
    body: JSON.stringify({ name }),
  });

export interface EditActionRequestBody {
  enabled?: boolean;
  selectedAdvisoryId?: string;
}

export const editAction = (actionId: string, body: EditActionRequestBody) =>
  fetchWithAuth<Policy>(`/api/v1/alerts/${actionId}/edit`, {
    method: "PUT",
    body: JSON.stringify(body),
  });

export const useAlerts = () =>
  useSWR<Action[]>("/api/v1/alerts", {
    refreshInterval: 10000,
    revalidateOnFocus: true,
  });

export const usePolicies = () =>
  useSWR<Policy[]>("/api/v1/policies", {
    revalidateOnFocus: true,
  });

export const usePolicy = (policyId: string | null) =>
  useSWR<Policy>(policyId ? `/api/v1/policies/${policyId}` : null);

export const useActionsForPolicy = (policyId: string | null) =>
  useSWR<Action[]>(policyId ? `/api/v1/policies/${policyId}/actions` : null, {
    revalidateOnFocus: true,
  });

export interface ReviewApply {
  Decision: "apply";
  RecommendationID: string;
}

export interface ReviewIgnore {
  Decision: "ignore";
}

export type AlertReview = ReviewIgnore | ReviewApply;

export const reviewAlert = (alertId: string, review: AlertReview) =>
  fetchWithAuth(`/api/v1/alerts/${alertId}/review`, {
    method: "POST",
    body: JSON.stringify(review),
  });
