// Thin fetch wrapper for /api/v1: sends the CSRF double-submit header on
// state-changing requests and maps problem+json errors (T016).
import type { Problem } from './types';

const BASE = '/api/v1';

/** ApiError carries the RFC 9457 problem body. */
export class ApiError extends Error {
  readonly status: number;
  readonly problem: Problem;

  constructor(problem: Problem) {
    super(problem.detail || problem.title || `HTTP ${problem.status}`);
    this.name = 'ApiError';
    this.status = problem.status;
    this.problem = problem;
  }
}

/** OfflineError marks a request that failed because the network is gone. */
export class OfflineError extends Error {
  constructor() {
    super('Sem conexão com a internet');
    this.name = 'OfflineError';
  }
}

function csrfToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)fluentdev_csrf=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : '';
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  let payload: BodyInit | undefined;

  if (body instanceof FormData) {
    payload = body;
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
    payload = JSON.stringify(body);
  }
  if (method !== 'GET') {
    headers['X-CSRF-Token'] = csrfToken();
  }

  // Fail fast when the device knows it is offline — waiting on a doomed
  // socket would block the outbox fallback (US5).
  if (!navigator.onLine) throw new OfflineError();

  let resp: Response;
  try {
    resp = await fetch(BASE + path, { method, headers, body: payload, credentials: 'same-origin' });
  } catch (err) {
    if (!navigator.onLine) throw new OfflineError();
    throw err;
  }

  if (!resp.ok) {
    let problem: Problem = { type: 'about:blank', title: resp.statusText, status: resp.status };
    try {
      problem = (await resp.json()) as Problem;
    } catch {
      // non-JSON error body — keep the fallback problem
    }
    throw new ApiError(problem);
  }
  if (resp.status === 204) return undefined as T;
  return (await resp.json()) as T;
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
};
