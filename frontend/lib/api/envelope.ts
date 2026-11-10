export interface ApiMeta {
  response_code: number;
  code: number;
  severity: "OK" | "WARNING" | "ERROR";
  message: string;
}

export type ApiSuccess<T> = { status: true; meta: ApiMeta } & T;

export interface ApiErrorEnvelope {
  status: false;
  meta: ApiMeta;
  messages: Record<string, string[]> | null;
}

export type ApiResult<T> =
  | { ok: true; httpStatus: number; data: ApiSuccess<T> }
  | {
      ok: false;
      httpStatus: number;
      meta: ApiMeta;
      messages: Record<string, string[]> | null;
      retryAfterSeconds: number | null;
    };

/** A request that never reached the API (DNS/connection failure, timeout, ...). */
export function networkFailureResult(): ApiResult<never> {
  return {
    ok: false,
    httpStatus: 0,
    meta: {
      response_code: 0,
      code: 0,
      severity: "ERROR",
      message: "Tidak ada koneksi internet. Periksa jaringan lalu coba lagi.",
    },
    messages: null,
    retryAfterSeconds: null,
  };
}

function parseRetryAfter(response: Response): number | null {
  const header = response.headers.get("Retry-After");
  if (!header) return null;
  const seconds = Number(header);
  return Number.isFinite(seconds) ? seconds : null;
}

export async function parseEnvelope<T>(response: Response): Promise<ApiResult<T>> {
  let body: unknown;
  try {
    body = await response.json();
  } catch {
    return {
      ok: false,
      httpStatus: response.status,
      meta: {
        response_code: response.status,
        code: response.status,
        severity: "ERROR",
        message: "Terjadi kesalahan pada server.",
      },
      messages: null,
      retryAfterSeconds: parseRetryAfter(response),
    };
  }

  const envelope = body as { status?: unknown; meta?: unknown; messages?: unknown };
  if (envelope.status === true) {
    return { ok: true, httpStatus: response.status, data: body as ApiSuccess<T> };
  }

  const meta = (envelope.meta as ApiMeta) ?? {
    response_code: response.status,
    code: response.status,
    severity: "ERROR",
    message: "Terjadi kesalahan pada server.",
  };
  return {
    ok: false,
    httpStatus: response.status,
    meta,
    messages: (envelope.messages as Record<string, string[]> | null) ?? null,
    retryAfterSeconds: parseRetryAfter(response),
  };
}
