import React, { useState, useEffect } from 'react';
import { Sidebar } from './components/layout/Sidebar';
import { Header } from './components/layout/Header';
import { Dashboard } from './pages/Dashboard';
import { ContainersPage } from './pages/Containers';
import { ContainerDetail } from './pages/ContainerDetail';
import { NetworkFlowPage } from './pages/NetworkFlow';
import { NetworksPage } from './pages/Networks';
import { ImagesPage } from './pages/Images';
import { VolumesPage } from './pages/Volumes';
import { GPUPage } from './pages/GPU';
import { EventsPage } from './pages/Events';
import { LogsPage } from './pages/Logs';
import { SettingsPage } from './pages/Settings';
import { SetupModal } from './components/auth/SetupModal';
import { LoginModal } from './components/auth/LoginModal';
import {
  AuthStatus,
  Container,
  HostMetrics,
  GPUTelemetry,
  SystemInfoResponse,
  Alert,
} from './types';
import { api, setCSRFToken } from './lib/api';
import { wsClient } from './lib/ws';

export const App: React.FC = () => {
  const [currentTab, setCurrentTab] = useState<string>('dashboard');
  const [selectedContainerId, setSelectedContainerId] = useState<string | null>(null);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  // App Data State
  const [authStatus, setAuthStatus] = useState<AuthStatus | undefined>(undefined);
  const [systemInfo, setSystemInfo] = useState<SystemInfoResponse | undefined>(undefined);
  const [hostMetrics, setHostMetrics] = useState<HostMetrics | undefined>(undefined);
  const [containers, setContainers] = useState<Container[]>([]);
  const [gpuTelemetry, setGpuTelemetry] = useState<GPUTelemetry | undefined>(undefined);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [wsConnected, setWsConnected] = useState(false);

  // Modals
  const [showSetupModal, setShowSetupModal] = useState(false);
  const [showLoginModal, setShowLoginModal] = useState(false);

  // Initial Data Fetch
  const refreshAll = async () => {
    try {
      const auth = await api.getAuthStatus();
      setAuthStatus(auth);
      if (auth.csrfToken) {
        setCSRFToken(auth.csrfToken);
      }
      if (auth.needsSetup) {
        setShowSetupModal(true);
        setShowLoginModal(false);
      } else if (!auth.authenticated) {
        setShowSetupModal(false);
        setShowLoginModal(true);
      } else {
        setShowSetupModal(false);
        setShowLoginModal(false);
      }

      const [sys, metrics, conts, gpu, al] = await Promise.all([
        api.getSystemInfo().catch(() => undefined),
        api.getSystemMetrics().catch(() => undefined),
        api.getContainers().catch(() => []),
        api.getGPU().catch(() => undefined),
        api.getAlerts(50).catch(() => []),
      ]);

      if (sys) setSystemInfo(sys);
      if (metrics) setHostMetrics(metrics);
      if (conts) setContainers(conts);
      if (gpu) setGpuTelemetry(gpu);
      if (al) setAlerts(al);
    } catch (err) {
      console.error('Failed to load initial data:', err);
    }
  };

  useEffect(() => {
    refreshAll();

    const unsubStatus = wsClient.onStatusChange((connected) => {
      setWsConnected(connected);
    });

    const unsubAlerts = wsClient.subscribe('alerts', (alert: Alert) => {
      setAlerts((prev) => [alert, ...prev.filter((a) => a.id !== alert.id)]);
    });

    return () => {
      unsubStatus();
      unsubAlerts();
    };
  }, []);

  const handleLogout = async () => {
    try {
      await api.logout();
      setCSRFToken(null);
      refreshAll();
    } catch (err) {
      console.error(err);
    }
  };

  // Header Title mapping
  const titles: Record<string, { title: string; subtitle: string }> = {
    dashboard: { title: 'Server Dashboard', subtitle: 'Zentrale Server- und Docker-Übersicht' },
    containers: { title: 'Container-Verwaltung', subtitle: 'Alle erkannten Container des CasaOS-Hosts' },
    'network-flow': { title: 'Network Flow', subtitle: 'Interaktive Netzwerktopologie und Live-Traffic' },
    networks: { title: 'Docker Netzwerke', subtitle: 'Subnetze, Treiber und verbundene Container' },
    images: { title: 'Docker Images', subtitle: 'Lokale Image-Registry und Speicherbelegung' },
    volumes: { title: 'Docker Volumes', subtitle: 'Persistente Datenspeicher und Mountpoints' },
    gpu: { title: 'GPU Telemetrie', subtitle: 'NVIDIA NVML & AMD/Intel Hardware-Monitoring' },
    events: { title: 'Docker Events', subtitle: 'Chronologischer System- und Lifecycle-Feed' },
    logs: { title: 'Container Logs', subtitle: 'Zentraler Log-Stream aller Container' },
    settings: { title: 'Einstellungen', subtitle: 'Schwellwerte, Abtastraten und Berechtigungen' },
  };

  const currentMeta = selectedContainerId
    ? { title: 'Container Details', subtitle: 'Detaillierte Metriken, Logs und Konfiguration' }
    : titles[currentTab] || { title: 'Orbit', subtitle: '' };

  const runningCount = containers.filter((c) => c.state === 'running').length;

  return (
    <div className="flex h-screen bg-[#070a12] text-slate-100 overflow-hidden font-sans">
      {/* Sidebar Navigation */}
      <Sidebar
        currentTab={selectedContainerId ? 'containers' : currentTab}
        setCurrentTab={(tab) => {
          setSelectedContainerId(null);
          setCurrentTab(tab);
        }}
        collapsed={sidebarCollapsed}
        setCollapsed={setSidebarCollapsed}
        authStatus={authStatus}
        dockerConnected={systemInfo?.dockerConnected ?? false}
        onLogout={handleLogout}
      />

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <Header
          title={currentMeta.title}
          subtitle={currentMeta.subtitle}
          wsConnected={wsConnected}
          alerts={alerts}
          runningContainersCount={runningCount}
          totalContainersCount={containers.length}
          onOpenAlerts={() => {
            setSelectedContainerId(null);
            setCurrentTab('events');
          }}
        />

        <main className="flex-1 overflow-y-auto p-6">
          {selectedContainerId ? (
            <ContainerDetail
              containerId={selectedContainerId}
              authStatus={authStatus}
              onBack={() => setSelectedContainerId(null)}
              onRefresh={refreshAll}
            />
          ) : currentTab === 'dashboard' ? (
            <Dashboard
              initialSystemInfo={systemInfo}
              initialMetrics={hostMetrics}
              initialContainers={containers}
              initialGPU={gpuTelemetry}
              onSelectContainer={(id) => setSelectedContainerId(id)}
            />
          ) : currentTab === 'containers' ? (
            <ContainersPage
              containers={containers}
              authStatus={authStatus}
              onSelectContainer={(id) => setSelectedContainerId(id)}
              onRefresh={refreshAll}
            />
          ) : currentTab === 'network-flow' ? (
            <NetworkFlowPage />
          ) : currentTab === 'networks' ? (
            <NetworksPage authStatus={authStatus} />
          ) : currentTab === 'images' ? (
            <ImagesPage authStatus={authStatus} />
          ) : currentTab === 'volumes' ? (
            <VolumesPage authStatus={authStatus} />
          ) : currentTab === 'gpu' ? (
            <GPUPage />
          ) : currentTab === 'events' ? (
            <EventsPage />
          ) : currentTab === 'logs' ? (
            <LogsPage containers={containers} />
          ) : currentTab === 'settings' ? (
            <SettingsPage authStatus={authStatus} />
          ) : null}
        </main>
      </div>

      {/* Initial Setup Modal */}
      <SetupModal
        isOpen={showSetupModal}
        onSuccess={() => {
          setShowSetupModal(false);
          refreshAll();
        }}
      />

      {/* Login Modal */}
      <LoginModal
        isOpen={showLoginModal}
        onSuccess={() => {
          setShowLoginModal(false);
          refreshAll();
        }}
      />
    </div>
  );
};

export default App;

