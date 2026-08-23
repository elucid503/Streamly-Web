export class ApiError extends Error {

  status: number;

  constructor(status: number, message: string) {

    super(message);

    this.status = status;

  }

}

const GET_REUSE_MS = 1000;
const MAX_RECENT_GETS = 128;

const inflightGets = new Map<string, Promise<unknown>>();

const recentGets = new Map<string, { expiresAt: number; value: unknown }>();

function rememberGet(path: string, value: unknown) {

  const now = Date.now();

  if (recentGets.size >= MAX_RECENT_GETS) {

    for (const [key, entry] of recentGets) {

      if (entry.expiresAt <= now || recentGets.size >= MAX_RECENT_GETS) {

        recentGets.delete(key);

      }

    }

  }

  recentGets.set(path, { value, expiresAt: now + GET_REUSE_MS });

}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {

  const method = (init?.method ?? "GET").toUpperCase();
  const canReuse = method === "GET" && !init?.body;

  if (canReuse) {

    const recent = recentGets.get(path);

    if (recent && recent.expiresAt > Date.now()) {

      return recent.value as T;

    }

    const inflight = inflightGets.get(path);

    if (inflight) {

      return inflight as Promise<T>;

    }

  }

  const promise = fetch(path, {

    credentials: "include",

    headers: {

      "Content-Type": "application/json",
      ...(init?.headers ?? {}),

    },

    ...init,

  }).then(async (res) => {

    if (res.status === 204) {

      return undefined as T;

    }

    const data = await res.json().catch(() => ({}));

    if (!res.ok) {

      throw new ApiError(res.status, data.error ?? "request failed");

    }

    return data as T;

  });

  if (!canReuse) {

    return promise;

  }

  inflightGets.set(path, promise);

  try {

    const data = await promise;

    rememberGet(path, data);

    return data;

  } finally {

    inflightGets.delete(path);

  }

}
