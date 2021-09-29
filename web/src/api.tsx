import useSWR from "swr";
import { Action, Finding, PolicyStatus, Token } from "./api-types";

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
  fetchWithAuth<Finding>(`/api/v1/actions/${actionId}/edit`, {
    method: "PUT",
    body: JSON.stringify(body),
  });

export const useActions = () => useSWR<Action[]>("/api/v1/actions");

export const useAction = (actionId: string | null) =>
  useSWR<Action>(actionId ? `/api/v1/actions/${actionId}` : null);

export const usePolicies = (status?: PolicyStatus) =>
  useSWR<Finding[]>(
    status ? `/api/v1/findings?status=${status}` : `/api/v1/findings`,
    {
      revalidateOnFocus: true,
    }
  );

export const useFinding = (findingId: string | null) =>
  useSWR<Finding>(findingId ? `/api/v1/findings/${findingId}` : null);

export const useActionsForPolicy = (findingId: string | null) =>
  useSWR<Action[]>(findingId ? `/api/v1/findings/${findingId}/actions` : null, {
    revalidateOnFocus: true,
  });

export const setPolicyStatus = (findingId: string, status: PolicyStatus) =>
  fetchWithAuth(`/api/v1/findings/${findingId}/status`, {
    method: "PUT",
    body: JSON.stringify({ status }),
  });
