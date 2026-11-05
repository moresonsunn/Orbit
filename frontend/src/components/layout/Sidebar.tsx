import React from 'react';
import {
  LayoutDashboard,
  Boxes,
  Network,
  Globe,
  Layers,
  HardDrive,
  Cpu,
  Activity,
  Terminal,
  Settings,
  ChevronLeft,
  ChevronRight,
  LogOut,
  ShieldAlert,
} from 'lucide-react';
import { cn } from '../../lib/utils';
import { AuthStatus } from '../../types';

interface SidebarProps {
  currentTab: string;
  setCurrentTab: (tab: string) => void;
  collapsed: boolean;
  setCollapsed: (c: boolean) => void;
  authStatus?: AuthStatus;
  dockerConnected: boolean;
  onLogout: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentTab,
  setCurrentTab,
  collapsed,
  setCollapsed,
  authStatus,
  dockerConnected,
  onLogout,
}) => {
  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { id: 'containers', label: 'Containers', icon: Boxes },
    { id: 'network-flow', label: 'Network Flow', icon: Network },
    { id: 'networks', label: 'Networks', icon: Globe },
    { id: 'images', label: 'Images', icon: Layers },
    { id: 'volumes', label: 'Volumes', icon: HardDrive },
    { id: 'gpu', label: 'GPU Monitoring', icon: Cpu },
    { id: 'events', label: 'Events', icon: Activity },
    { id: 'logs', label: 'Logs', icon: Terminal },
    { id: 'settings', label: 'Settings', icon: Settings },
  ];

  return (
    <aside
      className={cn(
        'relative flex flex-col h-screen bg-[#141414] border-r border-[#222222] transition-all duration-300 z-30 select-none font-sans',
        collapsed ? 'w-16' : 'w-56'
      )}
    >
      {/* Brand Header (Tailscale org style) */}
      <div className="flex items-center justify-between h-14 px-3.5 border-b border-[#222222]">
        <div className="flex items-center gap-2.5 overflow-hidden cursor-pointer" onClick={() => setCurrentTab('dashboard')}>
          <div className="relative flex items-center justify-center w-7 h-7 rounded-md bg-[#222222] text-neutral-300 font-bold text-xs shrink-0 border border-[#2e2e2e]">
            <span className="tracking-tighter">O</span>
            <span
              className={cn(
                'absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full border border-[#141414]',
                dockerConnected ? 'bg-emerald-500' : 'bg-rose-500 animate-pulse'
              )}
              title={dockerConnected ? 'Docker Daemon verbunden' : 'Docker Daemon getrennt'}
            />
          </div>
          {!collapsed && (
            <div className="flex items-center gap-1.5 truncate">
              <span className="font-semibold text-xs text-neutral-200 truncate">orbit.local</span>
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-[#222222] text-neutral-400 font-normal">
                Host
              </span>
            </div>
          )}
        </div>

        <button
          onClick={() => setCollapsed(!collapsed)}
          className="p-1 rounded text-neutral-500 hover:text-white hover:bg-white/[0.05] transition-colors"
          title={collapsed ? 'Sidebar ausklappen' : 'Sidebar einklappen'}
        >
          {collapsed ? <ChevronRight className="w-3.5 h-3.5" /> : <ChevronLeft className="w-3.5 h-3.5" />}
        </button>
      </div>

      {/* Navigation Links */}
      <nav className="flex-1 py-3 px-2 space-y-0.5 overflow-y-auto">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = currentTab === item.id;
          return (
            <button
              key={item.id}
              onClick={() => setCurrentTab(item.id)}
              className={cn(
                'flex items-center w-full gap-2.5 px-2.5 py-1.5 rounded-md text-[13px] transition-colors group',
                isActive
                  ? 'bg-white/[0.08] text-white font-medium'
                  : 'text-neutral-400 hover:text-neutral-200 hover:bg-white/[0.03]'
              )}
              title={collapsed ? item.label : undefined}
            >
              <Icon
                className={cn(
                  'w-4 h-4 shrink-0 transition-colors',
                  isActive ? 'text-white' : 'text-neutral-500 group-hover:text-neutral-300'
                )}
              />
              {!collapsed && <span className="truncate">{item.label}</span>}
            </button>
          );
        })}
      </nav>

      {/* Read-Only Mode Notice if actions disabled */}
      {!collapsed && !authStatus?.features.enableActions && (
        <div className="mx-2 mb-2 p-2 rounded-md bg-[#1a1a1a] border border-[#262626] text-[11px] text-neutral-400 flex items-start gap-2">
          <ShieldAlert className="w-3.5 h-3.5 text-neutral-400 shrink-0 mt-0.5" />
          <span>Monitoring-Modus (Read-Only)</span>
        </div>
      )}

      {/* User Footer (Tailscale user style) */}
      <div className="p-3 border-t border-[#222222] flex items-center justify-between">
        {!collapsed ? (
          <div className="flex items-center gap-2.5 overflow-hidden">
            <div className="w-7 h-7 rounded-full bg-[#242424] border border-[#2e2e2e] flex items-center justify-center text-xs font-medium text-neutral-300 shrink-0">
              {authStatus?.user?.username?.charAt(0).toUpperCase() || 'A'}
            </div>
            <div className="flex flex-col truncate">
              <span className="text-xs font-medium text-neutral-200 truncate">{authStatus?.user?.username || 'admin'}</span>
              <span className="text-[11px] text-neutral-500 truncate">{authStatus?.user?.username || 'admin'}@orbit.local</span>
            </div>
          </div>
        ) : (
          <div className="w-7 h-7 rounded-full bg-[#242424] border border-[#2e2e2e] flex items-center justify-center text-xs font-medium text-neutral-300 mx-auto">
            {authStatus?.user?.username?.charAt(0).toUpperCase() || 'A'}
          </div>
        )}

        {!collapsed && authStatus?.authenticated && (
          <button
            onClick={onLogout}
            className="p-1 rounded text-neutral-500 hover:text-rose-400 hover:bg-white/[0.05] transition-colors"
            title="Abmelden"
          >
            <LogOut className="w-3.5 h-3.5" />
          </button>
        )}
      </div>
    </aside>
  );
};

