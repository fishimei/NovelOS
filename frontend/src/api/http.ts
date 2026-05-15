import type { ApiErrorPayload, PaginatedResponse, StandardResponse } from '../types/api';

const API_BASE_URL = '/api/v1';

export class ApiError extends Error {
  code?: string;
  details?: unknown;
  requestId?: string;
  status: number;

  constructor(message: string, status: number, payload?: ApiErrorPayload) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = payload?.error?.code;
    this.details = payload?.error?.details;
    this.requestId = payload?.meta?.request_id;
  }
}

type RequestOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  body?: unknown;
  signal?: AbortSignal;
  headers?: HeadersInit;
};

async function parseJson(response: Response) {
  const text = await response.text();

  if (!text) {
    return undefined;
  }

  return JSON.parse(text);
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);

  if (options.body !== undefined) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: options.method ?? 'GET',
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    headers,
    signal: options.signal,
  });

  const payload = await parseJson(response);

  if (!response.ok) {
    const errorPayload = payload as ApiErrorPayload | undefined;
    throw new ApiError(
      errorPayload?.error?.message ?? `Request failed with status ${response.status}`,
      response.status,
      errorPayload,
    );
  }

  return payload as T;
}

export async function getData<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await request<StandardResponse<T>>(path, { signal });
  return response.data;
}

export async function getPage<T>(path: string, signal?: AbortSignal): Promise<PaginatedResponse<T>> {
  return request<PaginatedResponse<T>>(path, { signal });
}

export async function postData<TResponse, TBody>(path: string, body: TBody): Promise<TResponse> {
  const response = await request<StandardResponse<TResponse>>(path, {
    method: 'POST',
    body,
  });
  return response.data;
}

export async function putData<TResponse, TBody>(path: string, body: TBody): Promise<TResponse> {
  const response = await request<StandardResponse<TResponse>>(path, {
    method: 'PUT',
    body,
  });
  return response.data;
}

export function pageParams(page = 1, pageSize = 20) {
  return new URLSearchParams({
    page: String(page),
    page_size: String(pageSize),
  }).toString();
}
