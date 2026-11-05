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
        'relative flex flex-col h-screen bg-[#070b14] border-r border-slate-800/80 transition-all duration-300 z-30 select-none',
        collapsed ? 'w-16' : 'w-64'
      )}
    >
      {/* Brand Header */}
      <div className="flex items-center justify-between h-16 px-4 border-b border-slate-800/80">
        <div className="flex items-center gap-3 overflow-hidden cursor-pointer" onClick={() => setCurrentTab('dashboard')}>
          <div className="relative flex items-center justify-center w-8 h-8 rounded-lg bg-gradient-to-tr from-cyan-600 to-indigo-600 shadow-md shadow-cyan-900/20 text-white font-bold text-sm">
            O
            <span
              className={cn(
                'absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border-2 border-[#070b14]',
                dockerConnected ? 'bg-emerald-400' : 'bg-rose-500 animate-pulse'
              )}
              title={dockerConnected ? 'Docker Daemon verbunden' : 'Docker Daemon getrennt'}
            />
          </div>
          {!collapsed && (
            <div className="flex flex-col">
              <span className="font-bold text-base tracking-wide text-white font-mono">ORBIT</span>
              <span className="text-[10px] text-slate-400 font-mono -mt-1 tracking-wider uppercase">Infrastructure</span>
            </div>
          )}
        </div>

        <button
          onClick={() => setCollapsed(!collapsed)}
          className="p-1 rounded-md text-slate-400 hover:text-white hover:bg-slate-800/80 transition-colors"
          title={collapsed ? 'Sidebar ausklappen' : 'Sidebar einklappen'}
        >
          {collapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
        </button>
      </div>

      {/* Navigation Links */}
      <nav className="flex-1 py-4 px-2 space-y-1 overflow-y-auto">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = currentTab === item.id;
          return (
            <button
              key={item.id}
              onClick={() => setCurrentTab(item.id)}
              className={cn(
                'flex items-center w-full gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all group',
                isActive
                  ? 'bg-cyan-950/60 text-cyan-300 border border-cyan-800/60 shadow-sm'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50 border border-transparent'
              )}
              title={collapsed ? item.label : undefined}
            >
              <Icon
                className={cn(
                  'w-4 h-4 shrink-0 transition-transform group-hover:scale-110',
                  isActive ? 'text-cyan-400' : 'text-slate-400 group-hover:text-slate-200'
                )}
              />
              {!collapsed && <span className="truncate">{item.label}</span>}
            </button>
          );
        })}
      </nav>

      {/* Read-Only Mode Notice if actions disabled */}
      {!collapsed && !authStatus?.features.enableActions && (
        <div className="mx-3 mb-3 p-2.5 rounded-lg bg-slate-900/80 border border-slate-800 text-[11px] text-slate-400 flex items-start gap-2">
          <ShieldAlert className="w-4 h-4 text-cyan-400 shrink-0 mt-0.5" />
          <span>Monitoring-Modus aktiv. Aktionen deaktiviert.</span>
        </div>
      )}

      {/* User Footer */}
      <div className="p-3 border-t border-slate-800/80 flex items-center justify-between">
        {!collapsed ? (
          <div className="flex items-center gap-2 overflow-hidden">
            <div className="w-7 h-7 rounded-full bg-slate-800 border border-slate-700 flex items-center justify-center text-xs font-bold text-slate-300">
              {authStatus?.user?.username?.charAt(0).toUpperCase() || 'U'}
            </div>
            <div className="flex flex-col truncate">
              <span className="text-xs font-medium text-slate-200 truncate">{authStatus?.user?.username || 'Benutzer'}</span>
              <span className="text-[10px] text-slate-400 uppercase tracking-wider">{authStatus?.user?.role || 'Viewer'}</span>
            </div>
          </div>
        ) : (
          <div className="w-7 h-7 rounded-full bg-slate-800 border border-slate-700 flex items-center justify-center text-xs font-bold text-slate-300 mx-auto">
            {authStatus?.user?.username?.charAt(0).toUpperCase() || 'U'}
          </div>
        )}

        {!collapsed && authStatus?.authenticated && (
          <button
            onClick={onLogout}
            className="p-1.5 rounded-md text-slate-400 hover:text-rose-400 hover:bg-slate-800/60 transition-colors"
            title="Abmelden"
          >
            <LogOut className="w-4 h-4" />
          </button>
        )}
      </div>
    </aside>
  );
};

