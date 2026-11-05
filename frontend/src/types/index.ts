export type Availability =
  | 'ok'
  | 'unavailable'
  | 'no_permission'
  | 'unsupported'
  | 'source_down'
  | 'pending'
  | 'disabled';

export interface Gauge {
  value: number | null;
  status: Availability;
  reason?: string;
}

export interface ContainerPort {
  privatePort: number;
  publicPort?: number;
  protocol: string;
  hostIp?: string;
}

export interface ContainerNet {
  name: string;
  networkId: string;
  endpointId: string;
  ipAddress?: string;
  prefixLen?: number;
  ipv6Address?: string;
  gateway?: string;
  macAddress?: string;
  aliases?: string[];
}

export interface Mount {
  type: string;
  name?: string;
  source: string;
  destination: string;
  rw: boolean;
  driver?: string;
}

export interface Limits {
  memoryBytes: number;
  memorySwapBytes: number;
  cpus: number;
  cpuShares: number;
  cpusetCpus?: string;
  pidsLimit: number;
}

export interface ContainerStats {
  id: string;
  ts: number;
  cpuPercent: Gauge;
  cpuPercentOfLimit: Gauge;
  onlineCpus: number;
  cpuThrottledPercent: Gauge;
  memUsage: Gauge;
  memLimit: Gauge;
  memPercent: Gauge;
  memCache: Gauge;
  memSwap: Gauge;
  netRxBytes: Gauge;
  netTxBytes: Gauge;
  netRxBps: Gauge;
  netTxBps: Gauge;
  netRxPps: Gauge;
  netTxPps: Gauge;
  blkReadBytes: Gauge;
  blkWriteBytes: Gauge;
  blkReadBps: Gauge;
  blkWriteBps: Gauge;
  pids: Gauge;
}

export interface Container {
  id: string;
  shortId: string;
  name: string;
  image: string;
  imageRepo: string;
  imageTag: string;
  imageId: string;
  command: string;
  state: 'created' | 'running' | 'paused' | 'restarting' | 'removing' | 'exited' | 'dead';
  health: 'healthy' | 'unhealthy' | 'starting' | 'none';
  healthFailingStreak: number;
  created: string;
  startedAt?: string;
  finishedAt?: string;
  restartCount: number;
  exitCode: number;
  oomKilled: boolean;
  error?: string;
  restartPolicy: string;
  restartMaxRetry: number;
  composeProject?: string;
  composeService?: string;
  appGroup: string;
  appSource: 'compose' | 'casaos' | 'standalone';
  labels: Record<string, string>;
  ports: ContainerPort[];
  exposedPorts: string[];
  networkMode: string;
  networks: ContainerNet[];
  mounts: Mount[];
  limits: Limits;
  privileged: boolean;
  runtime?: string;
  gpuRequested: boolean;
  hasHealthcheck: boolean;
  isOrbit: boolean;
  stats?: ContainerStats;
}

export interface HostMetrics {
  timestamp: string;
  cpu: {
    modelName?: string;
    totalPercent: Gauge;
    cores: number;
    threads: number;
    perCore: Gauge[];
    freqMHz: Gauge;
    tempC: Gauge;
    loadAvg1: Gauge;
    loadAvg5: Gauge;
    loadAvg15: Gauge;
  };
  memory: {
    totalBytes: number;
    usedBytes: number;
    freeBytes: number;
    availableBytes: number;
    buffersBytes: number;
    cachedBytes: number;
    usedPercent: Gauge;
  };
  swap: {
    totalBytes: number;
    usedBytes: number;
    freeBytes: number;
    usedPercent: Gauge;
  };
  storage: {
    totalBytes: number;
    usedBytes: number;
    freeBytes: number;
    usedPercent: Gauge;
    readBps: Gauge;
    writeBps: Gauge;
    mounts: Array<{
      mountPoint: string;
      device: string;
      fsType: string;
      totalBytes: number;
      usedBytes: number;
      freeBytes: number;
      usedPercent: Gauge;
    }>;
  };
  network: {
    rxBps: Gauge;
    txBps: Gauge;
    totalRxBytes: number;
    totalTxBytes: number;
    interfaces: Array<{
      name: string;
      rxBytes: number;
      txBytes: number;
      rxBps: Gauge;
      txBps: Gauge;
    }>;
  };
  thermal?: {
    cpuTemp: Gauge;
    maxTemp: Gauge;
    sensors: Array<{
      name: string;
      label: string;
      device: string;
      tempC: Gauge;
      critC: Gauge;
      sensorType: string;
    }>;
  };
  system: {
    hostname: string;
    os: string;
    kernelVersion: string;
    architecture: string;
    uptimeSeconds: Gauge;
  };
}

export interface GPUProcessInfo {
  pid: number;
  containerId?: string;
  containerName?: string;
  processName: string;
  usedMemoryBytes: Gauge;
}

export interface GPUDevice {
  index: number;
  model: string;
  vendor: string;
  driverVersion: string;
  utilizationPercent: Gauge;
  memoryUsedBytes: Gauge;
  memoryTotalBytes: Gauge;
  memoryPercent: Gauge;
  temperatureCelsius: Gauge;
  powerDrawWatts: Gauge;
  powerLimitWatts: Gauge;
  clockGraphicsMHz: Gauge;
  clockMemoryMHz: Gauge;
  encoderUtilPercent: Gauge;
  decoderUtilPercent: Gauge;
  processes?: GPUProcessInfo[];
}

export interface GPUTelemetry {
  timestamp: string;
  available: boolean;
  message: string;
  devices?: GPUDevice[];
}

export interface TopologyNode {
  id: string;
  type: 'host' | 'network' | 'container' | 'gateway' | 'external';
  label: string;
  subnet?: string;
  ip?: string;
  status?: string;
  rxBps: Gauge;
  txBps: Gauge;
  metadata?: Record<string, any>;
}

export interface TopologyEdge {
  id: string;
  source: string;
  target: string;
  type: 'member' | 'traffic' | 'gateway';
  active: boolean;
  rxBps: Gauge;
  txBps: Gauge;
  protocol?: string;
}

export interface SocketConnection {
  protocol: string;
  localIp: string;
  localPort: number;
  remoteIp: string;
  remotePort: number;
  state: string;
  containerId?: string;
  containerName?: string;
  appProtocol?: string;
  classification: string;
  detectionMode: string;
}

export interface ProtocolSummary {
  protocol: string;
  configuredPortsCount: number;
  observedConnectionsCount: number;
  activeFlows: number;
}

export interface NetflowSnapshot {
  timestamp: string;
  ebpfAvailable: boolean;
  ebpfStatus: string;
  topology: {
    nodes: TopologyNode[];
    edges: TopologyEdge[];
  };
  connections: SocketConnection[];
  protocolStats: ProtocolSummary[];
}

export interface DockerEvent {
  id: number;
  timestamp: string;
  type: string;
  action: string;
  actorId: string;
  actorName: string;
  severity: 'info' | 'warning' | 'critical';
  attributes?: Record<string, string>;
}

export interface Alert {
  id: number;
  rule: string;
  target: string;
  targetName: string;
  severity: 'warning' | 'critical';
  message: string;
  value?: number;
  threshold?: number;
  startedAt: string;
  resolvedAt?: string;
  acknowledged: boolean;
}

export interface AuthStatus {
  needsSetup: boolean;
  authenticated: boolean;
  features: {
    enableActions: boolean;
    enableDestructive: boolean;
    ebpfMode: string;
    gpuMode: string;
  };
  user?: {
    id: number;
    username: string;
    role: 'admin' | 'viewer';
  };
  csrfToken?: string;
}

export interface SystemInfoResponse {
  dockerConnected: boolean;
  dockerError?: string;
  docker: {
    serverVersion: string;
    apiVersion: string;
    operatingSystem: string;
    kernelVersion: string;
    architecture: string;
    ncpu: number;
    memTotal: number;
    name: string;
    storageDriver: string;
    cgroupVersion: string;
    containers: number;
    containersRunning: number;
    containersPaused: number;
    containersStopped: number;
    images: number;
  };
  host: {
    hostname: string;
    os: string;
    kernelVersion: string;
    architecture: string;
    uptimeSeconds: Gauge;
  };
  inventoryCounts: {
    total: number;
    running: number;
    paused: number;
    restarting: number;
    stopped: number;
    unhealthy: number;
    healthy: number;
    images?: number;
    networks?: number;
    volumes?: number;
  };
}

export interface DockerNetwork {
  Name: string;
  Id: string;
  Created: string;
  Scope: string;
  Driver: string;
  EnableIPv4: boolean;
  EnableIPv6: boolean;
  Internal: boolean;
  IPAM: {
    Driver: string;
    Config?: Array<{
      Subnet?: string;
      Gateway?: string;
    }>;
  };
  Containers?: Record<string, any>;
}

export interface DockerImage {
  Id: string;
  RepoTags?: string[];
  RepoDigests?: string[];
  Size: number;
  Created: number;
  Containers: number;
}

export interface DockerVolume {
  name: string;
  driver: string;
  mountpoint: string;
  createdAt?: string;
  sizeBytes?: number;
  labels?: Record<string, string>;
}

export interface Settings {
  intervals: {
    hostMs: number;
    containerMs: number;
    gpuMs: number;
    netflowMs: number;
    historyStepSec: number;
  };
  retention: {
    rawHours: number;
    days5m: number;
    days1h: number;
    eventDays: number;
  };
  alerts: {
    hostCpu: { enabled: boolean; value: number; durationSec: number };
    hostMemory: { enabled: boolean; value: number; durationSec: number };
    diskUsage: { enabled: boolean; value: number; durationSec: number };
    containerCpu: { enabled: boolean; value: number; durationSec: number };
    containerMemory: { enabled: boolean; value: number; durationSec: number };
    networkRx: { enabled: boolean; value: number; durationSec: number };
    networkTx: { enabled: boolean; value: number; durationSec: number };
    gpuTemp: { enabled: boolean; value: number; durationSec: number };
    containerCrash: { enabled: boolean };
    restartLoop: { enabled: boolean; value: number; durationSec: number };
    containerUnhealthy: { enabled: boolean };
  };
  webhook: {
    enabled: boolean;
    url: string;
    minSeverity: 'info' | 'warning' | 'critical';
  };
}

