import useSWR from "swr";
import { Alert } from "./api-types";

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

export const checkAuthToken = async (token: string): Promise<boolean> => {
  return fetchWithAuth("/api/v1/login", { method: "POST" }, token)
    .then((res) => {
      if (res.status === 200) {
        console.log("aa");
        return true;
      }
      return false;
    })
    .catch(() => false);
};

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
