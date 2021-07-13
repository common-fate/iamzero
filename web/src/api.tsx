import useSWR from "swr";
import { Alert, Token } from "./api-types";

/**
 * Adds the x-iamzero-token header to auth requests.
 * You can provide a token as an argument, otherwise it fetches the token from
 * localStorage.
 * Only adds the header if a token is found.
 */
export const fetchWithAuth = (
  path: string,
  init?: RequestInit | undefined,
  token?: string
) => {
  const authToken = token ?? localStorage.getItem("iamzeroToken");
  const headers = authToken
    ? { ...init?.headers, "x-iamzero-token": authToken }
    : init?.headers;

  return fetch(path, {
    ...init,
    headers,
  });
};

export interface GetTokensResponse {
  tokens: Token[];
}

export const useTokens = () => useSWR<GetTokensResponse>("/api/v1/tokens");

export const deleteToken = (tokenId: string) =>
  fetchWithAuth(`/api/v1/tokens/${tokenId}`, {
    method: "DELETE",
    headers: {
      "Accept": "application/json",
      "Content-Type": "application/json",
    },
  });

export const createToken = (name: string) =>
  fetchWithAuth(`/api/v1/tokens`, {
    method: "POST",
    body: JSON.stringify({ name }),
    headers: {
      "Accept": "application/json",
      "Content-Type": "application/json",
    },
  });

export const useAlerts = () =>
  useSWR<Alert[]>("/api/v1/alerts", {
    refreshInterval: 10000,
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
    headers: {
      "Accept": "application/json",
      "Content-Type": "application/json",
    },
  });
