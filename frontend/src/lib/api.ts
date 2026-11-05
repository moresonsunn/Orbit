import {
  AuthStatus,
  Container,
  ContainerStats,
  DockerEvent,
  DockerImage,
  DockerNetwork,
  DockerVolume,
  GPUTelemetry,
  HostMetrics,
  NetflowSnapshot,
  Alert,
  Settings,
  SystemInfoResponse,
} from '../types';

let csrfToken: string | null = null;

export function setCSRFToken(token: string | null) {
  csrfToken = token;
}

export function getCSRFToken(): string | null {
  return csrfToken;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers || {});
  headers.set('Accept', 'application/json');

  if (options.body && typeof options.body === 'string' && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  // Inject CSRF token on modifying methods
  if (csrfToken && ['POST', 'PUT', 'DELETE', 'PATCH'].includes(options.method?.toUpperCase() || '')) {
    headers.set('X-CSRF-Token', csrfToken);
  }

  options.credentials = 'include';
  options.headers = headers;

  const res = await fetch(`/api/v1${path}`, options);

  if (!res.ok) {
    let errMsg = `Request failed: ${res.statusText}`;
    try {
      const errObj = await res.json();
      if (errObj && errObj.error) {
        errMsg = errObj.error;
      }
    } catch {
      // not json
    }
    throw new Error(errMsg);
  }

  return res.json();
}

export const api = {
  // Auth
  getAuthStatus: () => request<AuthStatus>('/auth/status'),
  setup: (data: { token: string; username: string; password: string }) =>
    request<{ message: string }>('/auth/setup', { method: 'POST', body: JSON.stringify(data) }),
  login: (data: { username: string; password: string }) =>
    request<{ user: any; csrfToken: string }>('/auth/login', { method: 'POST', body: JSON.stringify(data) }),
  logout: () => request<{ message: string }>('/auth/logout', { method: 'POST' }),

  // System & Host
  getSystemInfo: () => request<SystemInfoResponse>('/system'),
  getSystemMetrics: () => request<HostMetrics>('/system/metrics'),
  getMetricsHistory: (series: string, from?: number, to?: number, resolution?: string) => {
    const params = new URLSearchParams({ series });
    if (from) params.set('from', from.toString());
    if (to) params.set('to', to.toString());
    if (resolution) params.set('resolution', resolution);
    return request<{ key: string; points: Array<{ ts: number; avg: number; min: number; max: number }> }>(
      `/metrics/history?${params.toString()}`
    );
  },

  // Containers
  getContainers: () => request<Container[]>('/containers'),
  getContainer: (id: string) => request<{ container: Container; raw?: any; inspect?: any }>(`/containers/${id}`),
  getContainerStats: (id: string) => request<ContainerStats>(`/containers/${id}/stats`),
  getContainerStatsHistory: (id: string) => request<ContainerStats[]>(`/containers/${id}/stats/history`),
  getContainerLogs: async (id: string, tail = 200, timestamps = true) => {
    const res = await fetch(`/api/v1/containers/${id}/logs?tail=${tail}&timestamps=${timestamps}`, {
      credentials: 'include',
    });
    return res.text();
  },
  getContainerTop: (id: string) => request<{ Processes: string[][]; Titles: string[] }>(`/containers/${id}/top`),

  // Container Actions
  startContainer: (id: string) => request<{ message: string }>(`/containers/${id}/start`, { method: 'POST' }),
  stopContainer: (id: string) => request<{ message: string }>(`/containers/${id}/stop`, { method: 'POST' }),
  restartContainer: (id: string) => request<{ message: string }>(`/containers/${id}/restart`, { method: 'POST' }),
  pauseContainer: (id: string) => request<{ message: string }>(`/containers/${id}/pause`, { method: 'POST' }),
  unpauseContainer: (id: string) => request<{ message: string }>(`/containers/${id}/unpause`, { method: 'POST' }),
  removeContainer: (id: string, force = false) =>
    request<{ message: string }>(`/containers/${id}?force=${force}`, { method: 'DELETE' }),

  // Bulk Actions
  bulkStart: (ids: string[]) =>
    request<Array<{ id: string; success: boolean; error?: string }>>('/containers/bulk/start', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  bulkStop: (ids: string[]) =>
    request<Array<{ id: string; success: boolean; error?: string }>>('/containers/bulk/stop', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),
  bulkRestart: (ids: string[]) =>
    request<Array<{ id: string; success: boolean; error?: string }>>('/containers/bulk/restart', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    }),

  // Resources
  getNetworks: () => request<DockerNetwork[]>('/networks'),
  getNetwork: (id: string) => request<DockerNetwork>(`/networks/${id}`),
  removeNetwork: (id: string) => request<{ message: string }>(`/networks/${id}`, { method: 'DELETE' }),

  getImages: () => request<DockerImage[]>('/images'),
  removeImage: (id: string) => request<any>(`/images/${id}`, { method: 'DELETE' }),
  pruneImages: (all = false) => request<any>(`/images/prune?all=${all}`, { method: 'POST' }),

  getVolumes: () => request<DockerVolume[]>('/volumes'),
  removeVolume: (name: string) => request<{ message: string }>(`/volumes/${name}`, { method: 'DELETE' }),
  pruneVolumes: (all = false) => request<any>(`/volumes/prune?all=${all}`, { method: 'POST' }),

  // Netflow & GPU
  getNetflow: () => request<NetflowSnapshot>('/netflow'),
  getGPU: () => request<GPUTelemetry>('/gpu'),

  // Events & Alerts
  getEvents: (limit = 100) => request<DockerEvent[]>(`/events?limit=${limit}`),
  getAlerts: (limit = 50) => request<Alert[]>(`/alerts?limit=${limit}`),
  ackAlert: (id: number) => request<{ message: string }>(`/alerts/${id}/ack`, { method: 'POST' }),

  // Settings
  getSettings: () => request<Settings>('/settings'),
  updateSettings: (s: Settings) => request<Settings>('/settings', { method: 'PUT', body: JSON.stringify(s) }),
};
