export type User = {
  id: string;
  screen_name: string;
  is_icq: boolean;
  suspended_status?: string;
  is_bot?: boolean | null;
};

export type UserAccount = {
  id: string;
  screen_name: string;
  profile?: string;
  email_address?: string;
  reg_status?: number;
  confirmed?: boolean;
  is_icq: boolean;
  suspended_status?: string;
  is_bot?: boolean | null;
};

export type SessionInstance = {
  num: number;
  idle_seconds: number;
  is_away: boolean;
  away_message?: string;
  is_invisible: boolean;
  remote_addr?: string;
  remote_port?: number;
};

export type Session = {
  id: string;
  screen_name: string;
  online_seconds: number;
  is_away?: boolean;
  away_message?: string;
  idle_seconds?: number;
  is_invisible?: boolean;
  is_icq: boolean;
  instance_count?: number;
  instances?: SessionInstance[];
};

export type SessionList = {
  count: number;
  sessions: Session[];
};

export type ChatRoom = {
  name: string;
  create_time?: string;
  creator_id?: string;
  participants?: { id: string; screen_name: string }[];
};

export type VersionInfo = {
  version?: string;
  commit?: string;
  date?: string;
};

export type DirectoryCategory = {
  id: number;
  name: string;
};

export type DirectoryKeyword = {
  id: number;
  name: string;
  category_id?: number;
};

export type WebAPIKey = {
  dev_id: string;
  app_name: string;
  created_at?: string;
  is_active?: boolean;
  rate_limit?: number;
  allowed_origins?: string[];
  capabilities?: string[];
  dev_key?: string;
};

export type FeedbagBuddy = {
  name: string;
  item_id: number;
};

export type FeedbagGroup = {
  group_id: number;
  group_name: string;
  buddies: FeedbagBuddy[];
};

export type LinkedAccounts = {
  linked_accounts: string[];
};

export type ApiProbe = {
  method: string;
  path: string;
  ok: boolean;
  status: number;
  detail?: string;
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers || {});
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(`/api${path}`, { ...init, headers });

  if (!res.ok) {
    const text = (await res.text()).trim();
    throw new Error(text || `${res.status} ${res.statusText}`);
  }

  if (res.status === 204) return undefined as T;

  const contentType = res.headers.get("content-type") || "";
  if (!contentType.includes("application/json")) {
    return (await res.text()) as T;
  }

  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

export const api = {
  // users
  getUsers: () => request<User[]>("/user"),
  createUser: (screen_name: string, password: string) =>
    request<string>("/user", {
      method: "POST",
      body: JSON.stringify({ screen_name, password }),
    }),
  deleteUser: (screen_name: string) =>
    request<void>("/user", {
      method: "DELETE",
      body: JSON.stringify({ screen_name }),
    }),
  setPassword: (screen_name: string, password: string) =>
    request<void>("/user/password", {
      method: "PUT",
      body: JSON.stringify({ screen_name, password }),
    }),
  getAccount: (screenname: string) =>
    request<UserAccount>(`/user/${encodeURIComponent(screenname)}/account`),
  patchAccount: (
    screenname: string,
    body: { suspended_status?: string | null; is_bot?: boolean | null },
  ) =>
    request<void>(`/user/${encodeURIComponent(screenname)}/account`, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),
  getUserIconUrl: (screenname: string) =>
    `/api/user/${encodeURIComponent(screenname)}/icon`,
  getLinkedAccounts: (screenname: string) =>
    request<LinkedAccounts>(
      `/user/${encodeURIComponent(screenname)}/linked-account`,
    ),
  addLinkedAccount: (screenname: string, linked_screen_name: string) =>
    request<void>(`/user/${encodeURIComponent(screenname)}/linked-account`, {
      method: "POST",
      body: JSON.stringify({ linked_screen_name }),
    }),
  removeLinkedAccount: (screenname: string, linked: string) =>
    request<void>(
      `/user/${encodeURIComponent(screenname)}/linked-account/${encodeURIComponent(linked)}`,
      { method: "DELETE" },
    ),
  getIcqProfile: (screenname: string) =>
    request<Record<string, unknown>>(
      `/user/${encodeURIComponent(screenname)}/icq`,
    ),
  putIcqProfile: (screenname: string, body: Record<string, unknown>) =>
    request<void>(`/user/${encodeURIComponent(screenname)}/icq`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  loginCheck: async (screen_name: string, password: string) => {
    const token = btoa(`${screen_name}:${password}`);
    return request<string>("/user/login", {
      headers: { Authorization: `Basic ${token}` },
    });
  },

  // sessions
  getSessions: () => request<SessionList>("/session"),
  getSession: (screenname: string) =>
    request<SessionList>(`/session/${encodeURIComponent(screenname)}`),
  kickSession: (screenname: string) =>
    request<void>(`/session/${encodeURIComponent(screenname)}`, {
      method: "DELETE",
    }),

  // chat
  getPublicRooms: () => request<ChatRoom[]>("/chat/room/public"),
  createPublicRoom: (name: string) =>
    request<string>("/chat/room/public", {
      method: "POST",
      body: JSON.stringify({ name }),
    }),
  deletePublicRooms: (names: string[]) =>
    request<void>("/chat/room/public", {
      method: "DELETE",
      body: JSON.stringify({ names }),
    }),
  getPrivateRooms: () => request<ChatRoom[]>("/chat/room/private"),

  // messaging
  sendIM: (from: string, to: string, text: string) =>
    request<string>("/instant-message", {
      method: "POST",
      body: JSON.stringify({ from, to, text }),
    }),

  // system
  getVersion: () => request<VersionInfo>("/version"),

  // directory
  getDirectoryCategories: () =>
    request<DirectoryCategory[]>("/directory/category"),
  createDirectoryCategory: (name: string) =>
    request<DirectoryCategory>("/directory/category", {
      method: "POST",
      body: JSON.stringify({ name }),
    }),
  deleteDirectoryCategory: (id: number) =>
    request<void>(`/directory/category/${id}`, { method: "DELETE" }),
  getDirectoryKeywords: (categoryId: number) =>
    request<DirectoryKeyword[]>(`/directory/category/${categoryId}/keyword`),
  createDirectoryKeyword: (name: string, category_id: number) =>
    request<DirectoryKeyword>("/directory/keyword", {
      method: "POST",
      body: JSON.stringify({ name, category_id }),
    }),
  deleteDirectoryKeyword: (id: number) =>
    request<void>(`/directory/keyword/${id}`, { method: "DELETE" }),

  // webapi keys
  getWebApiKeys: () => request<WebAPIKey[]>("/admin/webapi/keys"),
  getWebApiKey: (id: string) =>
    request<WebAPIKey>(`/admin/webapi/keys/${encodeURIComponent(id)}`),
  createWebApiKey: (body: {
    app_name: string;
    allowed_origins?: string[];
    rate_limit?: number;
    capabilities?: string[];
  }) =>
    request<WebAPIKey>("/admin/webapi/keys", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updateWebApiKey: (
    id: string,
    body: Partial<{
      app_name: string;
      is_active: boolean;
      allowed_origins: string[];
      rate_limit: number;
      capabilities: string[];
    }>,
  ) =>
    request<WebAPIKey>(`/admin/webapi/keys/${encodeURIComponent(id)}`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  deleteWebApiKey: (id: string) =>
    request<void>(`/admin/webapi/keys/${encodeURIComponent(id)}`, {
      method: "DELETE",
    }),

  // feedbag / buddy list
  getFeedbag: (screen_name: string) =>
    request<FeedbagGroup[]>(
      `/feedbag/${encodeURIComponent(screen_name)}/group`,
    ),
  createFeedbagGroup: (screen_name: string, group_name: string) =>
    request<{ group_id: number; group_name: string }>(
      `/feedbag/${encodeURIComponent(screen_name)}/group/${encodeURIComponent(group_name)}`,
      { method: "PUT" },
    ),
  addBuddy: (
    screen_name: string,
    group_id: number,
    buddy_screen_name: string,
  ) =>
    request<void>(
      `/feedbag/${encodeURIComponent(screen_name)}/group/${group_id}/buddy/${encodeURIComponent(buddy_screen_name)}`,
      { method: "PUT" },
    ),
  removeBuddy: (
    screen_name: string,
    group_id: number,
    buddy_screen_name: string,
  ) =>
    request<void>(
      `/feedbag/${encodeURIComponent(screen_name)}/group/${group_id}/buddy/${encodeURIComponent(buddy_screen_name)}`,
      { method: "DELETE" },
    ),

  // bart assets
  listBart: (type = 1) => request<unknown[]>(`/bart?type=${type}`),
  getBartUrl: (hash: string) => `/api/bart/${encodeURIComponent(hash)}`,
  deleteBart: (hash: string) =>
    request<void>(`/bart/${encodeURIComponent(hash)}`, { method: "DELETE" }),

  // health probe of major GET endpoints
  probeApis: async (): Promise<ApiProbe[]> => {
    const checks: { method: string; path: string }[] = [
      { method: "GET", path: "/version" },
      { method: "GET", path: "/user" },
      { method: "GET", path: "/session" },
      { method: "GET", path: "/chat/room/public" },
      { method: "GET", path: "/chat/room/private" },
      { method: "GET", path: "/directory/category" },
      { method: "GET", path: "/admin/webapi/keys" },
      { method: "GET", path: "/bart?type=1" },
    ];

    return Promise.all(
      checks.map(async (c) => {
        try {
          const res = await fetch(`/api${c.path}`);
          return {
            method: c.method,
            path: c.path,
            ok: res.ok,
            status: res.status,
            detail: res.ok ? "ok" : (await res.text()).slice(0, 120),
          };
        } catch (e) {
          return {
            method: c.method,
            path: c.path,
            ok: false,
            status: 0,
            detail: e instanceof Error ? e.message : "network error",
          };
        }
      }),
    );
  },
};
