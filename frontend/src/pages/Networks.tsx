import React, { useState, useEffect, useMemo } from 'react';
import {
  Network,
  Globe,
  Server,
  Layers,
  Shield,
  ShieldCheck,
  Trash2,
  Search,
  Copy,
  Check,
  ChevronDown,
  ChevronUp,
  RefreshCw,
  X,
  Activity,
  Lock,
  Unlock,
  Radio,
  SlidersHorizontal,
  Info,
  Boxes,
} from 'lucide-react';
import { DockerNetwork, AuthStatus, NetworkContainerEndpoint } from '../types';
import { api } from '../lib/api';
import { ActionConfirmModal } from '../components/modals/ActionConfirmModal';
import { StatusBadge } from '../components/common/StatusBadge';
import { MetricCard } from '../components/common/MetricCard';
import { cn } from '../lib/utils';

interface NetworksPageProps {
  authStatus?: AuthStatus;
}

type FilterDriver = 'all' | 'bridge' | 'overlay' | 'macvlan' | 'host' | 'internal';

export const NetworksPage: React.FC<NetworksPageProps> = ({ authStatus }) => {
  const [networks, setNetworks] = useState<DockerNetwork[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [deleteModal, setDeleteModal] = useState<DockerNetwork | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeFilter, setActiveFilter] = useState<FilterDriver>('all');
  const [expandedCards, setExpandedCards] = useState<Set<string>>(new Set());
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  const destructiveEnabled = authStatus?.features.enableDestructive ?? false;

  const loadNetworks = async (isManualRefresh = false) => {
    if (isManualRefresh) setRefreshing(true);
    else setLoading(true);

    try {
      const data = await api.getNetworks();
      setNetworks(data);
      // Auto-expand all cards on initial load so IPAM and container clients are immediately visible
      setExpandedCards((prev) => {
        if (prev.size === 0) {
          return new Set(data.map((n) => n.Id));
        }
        return prev;
      });
    } catch (err) {
      console.error('Failed to load networks:', err);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  useEffect(() => {
    loadNetworks();
  }, []);

  const handleDelete = async () => {
    if (!deleteModal) return;
    try {
      await api.removeNetwork(deleteModal.Id);
      setDeleteModal(null);
      loadNetworks(true);
    } catch (err: any) {
      alert(`Fehler beim Entfernen des Netzwerks: ${err.message}`);
    }
  };

  const copyToClipboard = async (text: string, key: string, e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    try {
      await navigator.clipboard.writeText(text);
      setCopiedKey(key);
      setTimeout(() => {
        setCopiedKey((curr) => (curr === key ? null : curr));
      }, 2000);
    } catch (err) {
      console.error('Copy failed:', err);
    }
  };

  const toggleCard = (id: string) => {
    setExpandedCards((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const expandAll = () => {
    setExpandedCards(new Set(networks.map((n) => n.Id)));
  };

  const collapseAll = () => {
    setExpandedCards(new Set());
  };

  // KPI Calculations
  const kpiStats = useMemo(() => {
    const totalNetworks = networks.length;
    let activeSubnets = 0;
    let totalContainerIPs = 0;
    let totalCapacity = 0;
    let totalUsed = 0;

    networks.forEach((net) => {
      if (net.SubnetDetails?.Subnet || (net.IPAM?.Config && net.IPAM.Config.length > 0 && net.IPAM.Config[0]?.Subnet)) {
        activeSubnets++;
      }
      const cCount = net.Containers ? Object.keys(net.Containers).length : 0;
      totalContainerIPs += cCount;

      if (net.SubnetDetails && net.SubnetDetails.TotalIPs > 0) {
        totalCapacity += net.SubnetDetails.TotalIPs;
        totalUsed += net.SubnetDetails.UsedIPs;
      }
    });

    const overallUtilization = totalCapacity > 0 ? (totalUsed / totalCapacity) * 100 : 0;

    return {
      totalNetworks,
      activeSubnets,
      totalContainerIPs,
      totalCapacity,
      totalUsed,
      overallUtilization,
    };
  }, [networks]);

  // Filter Counts
  const filterCounts = useMemo(() => {
    const counts = {
      all: networks.length,
      bridge: 0,
      overlay: 0,
      macvlan: 0,
      host: 0,
      internal: 0,
    };

    networks.forEach((n) => {
      const d = n.Driver?.toLowerCase() || '';
      if (d === 'bridge') counts.bridge++;
      else if (d === 'overlay') counts.overlay++;
      else if (d === 'macvlan' || d === 'ipvlan') counts.macvlan++;
      else if (d === 'host') counts.host++;

      if (n.Internal) counts.internal++;
    });

    return counts;
  }, [networks]);

  // Filtered networks based on search and driver filter
  const filteredNetworks = useMemo(() => {
    return networks.filter((net) => {
      if (activeFilter === 'internal') {
        if (!net.Internal) return false;
      } else if (activeFilter !== 'all') {
        const d = net.Driver?.toLowerCase() || '';
        if (activeFilter === 'macvlan') {
          if (d !== 'macvlan' && d !== 'ipvlan') return false;
        } else if (d !== activeFilter) {
          return false;
        }
      }

      if (!searchQuery.trim()) return true;
      const q = searchQuery.toLowerCase().trim();

      if (net.Name.toLowerCase().includes(q)) return true;
      if (net.Driver.toLowerCase().includes(q)) return true;
      if (net.Id.toLowerCase().includes(q)) return true;
      if (net.SubnetDetails?.Subnet?.toLowerCase().includes(q)) return true;
      if (net.SubnetDetails?.Gateway?.toLowerCase().includes(q)) return true;
      if (net.IPAM?.Config?.some((c) => c.Subnet?.toLowerCase().includes(q) || c.Gateway?.toLowerCase().includes(q))) return true;

      if (net.Containers) {
        for (const c of Object.values(net.Containers)) {
          if (c.Name?.toLowerCase().includes(q)) return true;
          if (c.IPv4Address?.toLowerCase().includes(q)) return true;
          if (c.IPv6Address?.toLowerCase().includes(q)) return true;
          if (c.MacAddress?.toLowerCase().includes(q)) return true;
          if (c.EndpointID?.toLowerCase().includes(q)) return true;
        }
      }

      return false;
    });
  }, [networks, activeFilter, searchQuery]);

  const getDriverBadgeClass = (driver: string) => {
    const d = driver.toLowerCase();
    if (d === 'bridge') return 'bg-blue-950/40 text-blue-400 border-blue-800/40';
    if (d === 'overlay') return 'bg-amber-950/40 text-amber-400 border-amber-800/40';
    if (d === 'macvlan' || d === 'ipvlan') return 'bg-purple-950/40 text-purple-400 border-purple-800/40';
    if (d === 'host') return 'bg-cyan-950/40 text-cyan-400 border-cyan-800/40';
    return 'bg-[#222222] text-neutral-300 border-[#2e2e2e]';
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-200 font-sans">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h2 className="text-2xl font-semibold text-white tracking-tight">Networks</h2>
            <span className="px-2 py-0.5 rounded-full text-[11px] font-mono font-medium bg-[#1c1c1c] text-neutral-400 border border-[#2a2a2a]">
              UniFi IPAM
            </span>
          </div>
          <p className="text-xs sm:text-sm text-neutral-400 mt-1">
            Docker-Netzwerktreiber, IPAM-Subnetze, Allokationspools und verbundene Container-Clients.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => loadNetworks(true)}
            disabled={refreshing || loading}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-[#181818] border border-[#282828] text-neutral-300 hover:text-white hover:border-[#383838] transition-colors disabled:opacity-50"
            title="Netzwerke neu laden"
          >
            <RefreshCw className={cn('w-3.5 h-3.5', refreshing && 'animate-spin text-blue-400')} />
            <span>Aktualisieren</span>
          </button>

          <div className="flex items-center rounded-md border border-[#282828] bg-[#181818] p-0.5">
            <button
              onClick={expandAll}
              className="px-2.5 py-1 text-xs text-neutral-400 hover:text-white transition-colors"
              title="Alle Details ausklappen"
            >
              Alle aufklappen
            </button>
            <span className="text-neutral-600">|</span>
            <button
              onClick={collapseAll}
              className="px-2.5 py-1 text-xs text-neutral-400 hover:text-white transition-colors"
              title="Alle Details einklappen"
            >
              Zuklappen
            </button>
          </div>
        </div>
      </div>

      {/* Top KPI Summary Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Gesamte Netzwerke"
          formattedValue={String(kpiStats.totalNetworks)}
          subtitle={`${filterCounts.bridge} Bridge · ${filterCounts.host} Host · ${filterCounts.internal} Intern`}
          icon={<Network className="w-4 h-4 text-blue-400" />}
        />
        <MetricCard
          title="Aktive IPAM-Subnetze"
          formattedValue={String(kpiStats.activeSubnets)}
          subtitle="Verwaltete IPv4/IPv6 Adressräume"
          icon={<Layers className="w-4 h-4 text-purple-400" />}
        />
        <MetricCard
          title="Zugewiesene Container-IPs"
          formattedValue={String(kpiStats.totalContainerIPs)}
          subtitle="Aktive Client-Verbindungen"
          icon={<Server className="w-4 h-4 text-emerald-400" />}
        />
        <MetricCard
          title="IP-Pool Gesamtauslastung"
          formattedValue={`${kpiStats.overallUtilization.toFixed(1)}%`}
          subtitle={`${kpiStats.totalUsed.toLocaleString()} / ${kpiStats.totalCapacity.toLocaleString()} IPs belegt`}
          progressPercent={kpiStats.overallUtilization}
          icon={<Activity className="w-4 h-4 text-cyan-400" />}
        />
      </div>

      {/* Search & Filter Toolbar */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 p-3 rounded-lg bg-[#181818] border border-[#282828]">
        {/* Search Input */}
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-neutral-500" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Netzwerk, CIDR, Container oder IP durchsuchen..."
            className="w-full pl-9 pr-8 py-1.5 bg-[#121212] border border-[#282828] rounded-md text-xs text-white placeholder-neutral-500 focus:outline-hidden focus:border-blue-500 transition-colors font-sans"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-neutral-500 hover:text-white"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}
        </div>

        {/* Filter Pills */}
        <div className="flex items-center gap-1.5 overflow-x-auto pb-1 md:pb-0 text-xs">
          {(
            [
              { key: 'all', label: 'Alle', count: filterCounts.all },
              { key: 'bridge', label: 'Bridge', count: filterCounts.bridge },
              { key: 'overlay', label: 'Overlay', count: filterCounts.overlay },
              { key: 'macvlan', label: 'Macvlan', count: filterCounts.macvlan },
              { key: 'host', label: 'Host', count: filterCounts.host },
              { key: 'internal', label: 'Intern', count: filterCounts.internal },
            ] as const
          ).map((filter) => {
            const isActive = activeFilter === filter.key;
            return (
              <button
                key={filter.key}
                onClick={() => setActiveFilter(filter.key)}
                className={cn(
                  'flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-medium transition-colors shrink-0',
                  isActive
                    ? 'bg-blue-600 text-white'
                    : 'bg-[#121212] border border-[#282828] text-neutral-400 hover:text-white hover:border-[#383838]'
                )}
              >
                <span>{filter.label}</span>
                <span
                  className={cn(
                    'px-1.5 py-0.2 rounded-full text-[10px] font-mono',
                    isActive ? 'bg-blue-700 text-white' : 'bg-[#202020] text-neutral-400'
                  )}
                >
                  {filter.count}
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Network Cards List */}
      {loading ? (
        <div className="grid grid-cols-1 gap-4">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="p-5 rounded-xl bg-[#181818] border border-[#282828] animate-pulse space-y-4"
            >
              <div className="h-6 bg-[#222222] rounded-md w-1/4" />
              <div className="h-16 bg-[#202020] rounded-md w-full" />
              <div className="h-24 bg-[#1e1e1e] rounded-md w-full" />
            </div>
          ))}
        </div>
      ) : filteredNetworks.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 px-4 rounded-xl bg-[#181818] border border-[#282828] text-center">
          <Network className="w-10 h-10 text-neutral-600 mb-3" />
          <h3 className="text-base font-semibold text-white">Keine Netzwerke gefunden</h3>
          <p className="text-xs text-neutral-400 mt-1 max-w-sm">
            {searchQuery
              ? `Keine Suchergebnisse für "${searchQuery}" mit dem gewählten Filter.`
              : 'Es wurden keine Docker-Netzwerke für diese Kategorie gefunden.'}
          </p>
          {(searchQuery || activeFilter !== 'all') && (
            <button
              onClick={() => {
                setSearchQuery('');
                setActiveFilter('all');
              }}
              className="mt-4 px-3.5 py-1.5 text-xs font-medium rounded-md bg-blue-600 hover:bg-blue-500 text-white transition-colors"
            >
              Filter zurücksetzen
            </button>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4">
          {filteredNetworks.map((net) => {
            const isExpanded = expandedCards.has(net.Id);
            const containerMap = net.Containers || {};
            const containerEntries = Object.entries(containerMap);
            const containerCount = containerEntries.length;
            const subnetDetails = net.SubnetDetails;
            const fallbackSubnet = net.IPAM?.Config?.[0]?.Subnet;
            const fallbackGateway = net.IPAM?.Config?.[0]?.Gateway;

            // Options extraction
            const mtu = net.Options?.['com.docker.network.driver.mtu'] || net.Options?.mtu;
            const icc = net.Options?.['com.docker.network.bridge.enable_icc'];
            const masquerade = net.Options?.['com.docker.network.bridge.enable_ip_masquerade'];
            const bridgeName = net.Options?.['com.docker.network.bridge.name'];

            // Compose labels extraction
            const composeProject = net.Labels?.['com.docker.compose.project'];
            const composeNetwork = net.Labels?.['com.docker.compose.network'];

            return (
              <div
                key={net.Id}
                className="rounded-xl bg-[#181818] border border-[#282828] hover:border-[#383838] transition-all duration-150 overflow-hidden"
              >
                {/* Network Card Header */}
                <div
                  onClick={() => toggleCard(net.Id)}
                  className="p-4 sm:p-5 flex flex-col md:flex-row md:items-center justify-between gap-3 cursor-pointer select-none bg-[#181818] hover:bg-[#1a1a1a] transition-colors"
                >
                  <div className="flex items-start sm:items-center gap-3 flex-wrap">
                    <div className="p-2 rounded-lg bg-[#141414] border border-[#282828] text-neutral-300">
                      <Network className="w-5 h-5 text-blue-400" />
                    </div>

                    <div>
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="text-base font-semibold text-white tracking-tight">
                          {net.Name}
                        </span>

                        {/* Driver Badge */}
                        <span
                          className={cn(
                            'px-2 py-0.5 rounded text-[11px] font-mono font-medium border',
                            getDriverBadgeClass(net.Driver)
                          )}
                        >
                          {net.Driver}
                        </span>

                        {/* Scope Badge */}
                        <span className="px-2 py-0.5 rounded text-[11px] font-mono text-neutral-400 bg-[#1c1c1c] border border-[#282828]">
                          Scope: {net.Scope}
                        </span>

                        {/* Internal Badge */}
                        {net.Internal && (
                          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-emerald-950/40 text-emerald-400 border border-emerald-800/40">
                            <Shield className="w-3 h-3" />
                            <span>Intern (Isoliert)</span>
                          </span>
                        )}

                        {/* IPv6 Badge */}
                        {net.EnableIPv6 && (
                          <span className="px-2 py-0.5 rounded text-[11px] font-mono text-purple-300 bg-purple-950/40 border border-purple-800/40">
                            IPv6
                          </span>
                        )}
                      </div>

                      <div className="flex items-center gap-2 mt-1 text-[11px] text-neutral-500 font-mono">
                        <span>ID: {net.Id.slice(0, 12)}</span>
                        <button
                          onClick={(e) => copyToClipboard(net.Id, `${net.Id}-id`, e)}
                          className="hover:text-neutral-300 transition-colors"
                          title="Vollständige ID kopieren"
                        >
                          {copiedKey === `${net.Id}-id` ? (
                            <Check className="w-3 h-3 text-emerald-400" />
                          ) : (
                            <Copy className="w-3 h-3" />
                          )}
                        </button>
                        {bridgeName && (
                          <>
                            <span>·</span>
                            <span className="text-neutral-400">Bridge: {bridgeName}</span>
                          </>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-3 self-end md:self-center">
                    {/* Container Count Pill */}
                    <span className="flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-mono font-medium bg-[#141414] border border-[#282828] text-neutral-300">
                      <Boxes className="w-3.5 h-3.5 text-neutral-500" />
                      <span>
                        {containerCount} {containerCount === 1 ? 'Container' : 'Container'}
                      </span>
                    </span>

                    {/* Delete Action Button */}
                    {destructiveEnabled && !['bridge', 'host', 'none'].includes(net.Name) && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteModal(net);
                        }}
                        className="p-1.5 rounded-md text-neutral-500 hover:text-rose-400 hover:bg-rose-950/30 hover:border hover:border-rose-900/50 transition-colors"
                        title="Netzwerk löschen"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}

                    {/* Expand/Collapse Chevron */}
                    <div className="p-1 text-neutral-500">
                      {isExpanded ? (
                        <ChevronUp className="w-4 h-4 text-neutral-400" />
                      ) : (
                        <ChevronDown className="w-4 h-4 text-neutral-400" />
                      )}
                    </div>
                  </div>
                </div>

                {/* Expandable Inspection Body */}
                {isExpanded && (
                  <div className="px-4 pb-5 sm:px-5 sm:pb-5 space-y-4 pt-2 border-t border-[#242424] bg-[#161616]/60">
                    {/* IPAM Subnet Inspection Section */}
                    {subnetDetails ? (
                      <div className="p-4 rounded-lg bg-[#141414] border border-[#242424] space-y-3">
                        <div className="flex items-center justify-between text-xs">
                          <span className="font-semibold text-white tracking-wide flex items-center gap-1.5">
                            <Globe className="w-3.5 h-3.5 text-blue-400" />
                            <span>IPAM Subnetz-Inspektion</span>
                          </span>
                          <span className="text-[11px] font-mono text-neutral-500">
                            Treiber: {net.IPAM?.Driver || 'default'}
                          </span>
                        </div>

                        {/* IPAM Metrics Grid */}
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2.5 font-mono text-xs">
                          {/* Subnet CIDR */}
                          <div className="p-2.5 rounded bg-[#181818] border border-[#262626]">
                            <span className="text-[10px] uppercase font-sans tracking-wider text-neutral-500 block mb-1">
                              Subnetz CIDR
                            </span>
                            <div className="flex items-center justify-between">
                              <span className="text-white font-medium">{subnetDetails.Subnet}</span>
                              <button
                                onClick={(e) => copyToClipboard(subnetDetails.Subnet, `${net.Id}-cidr`, e)}
                                className="text-neutral-500 hover:text-white transition-colors"
                                title="CIDR kopieren"
                              >
                                {copiedKey === `${net.Id}-cidr` ? (
                                  <Check className="w-3 h-3 text-emerald-400" />
                                ) : (
                                  <Copy className="w-3 h-3" />
                                )}
                              </button>
                            </div>
                          </div>

                          {/* Gateway IP */}
                          <div className="p-2.5 rounded bg-[#181818] border border-[#262626]">
                            <span className="text-[10px] uppercase font-sans tracking-wider text-neutral-500 block mb-1">
                              Standard-Gateway
                            </span>
                            <div className="flex items-center justify-between">
                              <span className="text-neutral-200">
                                {subnetDetails.Gateway || '—'}
                              </span>
                              {subnetDetails.Gateway && (
                                <button
                                  onClick={(e) => copyToClipboard(subnetDetails.Gateway, `${net.Id}-gw`, e)}
                                  className="text-neutral-500 hover:text-white transition-colors"
                                  title="Gateway kopieren"
                                >
                                  {copiedKey === `${net.Id}-gw` ? (
                                    <Check className="w-3 h-3 text-emerald-400" />
                                  ) : (
                                    <Copy className="w-3 h-3" />
                                  )}
                                </button>
                              )}
                            </div>
                          </div>

                          {/* Usable Host Range */}
                          <div className="p-2.5 rounded bg-[#181818] border border-[#262626]">
                            <span className="text-[10px] uppercase font-sans tracking-wider text-neutral-500 block mb-1">
                              Nutzbarer Host-Bereich
                            </span>
                            <span className="text-neutral-300 block truncate" title={subnetDetails.UsableRange}>
                              {subnetDetails.UsableRange || '—'}
                            </span>
                          </div>

                          {/* Network & Broadcast IP */}
                          <div className="p-2.5 rounded bg-[#181818] border border-[#262626]">
                            <span className="text-[10px] uppercase font-sans tracking-wider text-neutral-500 block mb-1">
                              Netzwerk / Broadcast
                            </span>
                            <div className="text-[11px] text-neutral-400 truncate">
                              <span>{subnetDetails.NetworkIP}</span>
                              <span className="text-neutral-600 mx-1">/</span>
                              <span>{subnetDetails.BroadcastIP}</span>
                            </div>
                          </div>
                        </div>

                        {/* UniFi-Style Visual IP Allocation Pool Progress Bar */}
                        <div className="pt-2 border-t border-[#202020] space-y-1.5">
                          <div className="flex items-center justify-between text-xs font-mono">
                            <span className="text-neutral-400 font-sans flex items-center gap-1.5">
                              <Activity className="w-3.5 h-3.5 text-neutral-500" />
                              <span>IP-Adresspool Belegung:</span>
                            </span>
                            <div className="flex items-center gap-2">
                              <span className="text-neutral-200 font-medium">
                                {subnetDetails.UsedIPs} / {subnetDetails.TotalIPs.toLocaleString()} IPs belegt (
                                {subnetDetails.UtilizationPercent.toFixed(1)}%)
                              </span>
                              <span className="text-neutral-600">·</span>
                              <span className="text-emerald-400 font-medium">
                                {subnetDetails.FreeIPs.toLocaleString()} frei
                              </span>
                            </div>
                          </div>

                          {/* Progress Bar with Color Warning Ramp */}
                          <div className="w-full bg-[#262626] h-2 rounded-full overflow-hidden">
                            <div
                              className={cn(
                                'h-full rounded-full transition-all duration-300',
                                subnetDetails.UtilizationPercent > 90
                                  ? 'bg-rose-500'
                                  : subnetDetails.UtilizationPercent > 70
                                  ? 'bg-amber-500'
                                  : 'bg-emerald-500'
                              )}
                              style={{
                                width: `${Math.min(100, Math.max(1, subnetDetails.UtilizationPercent))}%`,
                              }}
                            />
                          </div>
                        </div>
                      </div>
                    ) : fallbackSubnet ? (
                      /* Fallback Subnet Display when SubnetDetails is absent */
                      <div className="p-3.5 rounded-lg bg-[#141414] border border-[#242424] space-y-2">
                        <div className="flex items-center justify-between text-xs">
                          <span className="font-semibold text-white tracking-wide">IPAM-Konfiguration</span>
                          <span className="text-neutral-400 font-mono text-[11px]">{fallbackSubnet}</span>
                        </div>
                        <div className="flex gap-4 text-xs font-mono text-neutral-300">
                          <div>Subnetz: <span className="text-white">{fallbackSubnet}</span></div>
                          {fallbackGateway && <div>Gateway: <span className="text-white">{fallbackGateway}</span></div>}
                        </div>
                      </div>
                    ) : (
                      /* Informational notice for host/none networks without IPAM */
                      <div className="p-3 rounded-lg bg-[#141414] border border-[#242424] flex items-center gap-2.5 text-xs text-neutral-400">
                        <Info className="w-4 h-4 text-neutral-500 shrink-0" />
                        <span>
                          {net.Driver === 'host'
                            ? 'Host-Treiber: Container nutzen den Netzwerk-Stack des Host-Systems direkt ohne separates Docker-Subnetz.'
                            : net.Driver === 'none'
                            ? 'None-Treiber: Vollständig isolierter Loopback-Namespace ohne externe Schnittstellen.'
                            : 'Kein separates IPAM-Subnetz für dieses Netzwerk konfiguriert.'}
                        </span>
                      </div>
                    )}

                    {/* Connected Container Client Table */}
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-semibold text-white tracking-wide flex items-center gap-1.5">
                          <Server className="w-3.5 h-3.5 text-emerald-400" />
                          <span>Verbundene Container-Clients ({containerCount})</span>
                        </span>
                        {containerCount > 0 && (
                          <span className="text-[11px] font-mono text-neutral-500">
                            DHCP / Statische Allokationen
                          </span>
                        )}
                      </div>

                      {containerCount === 0 ? (
                        <div className="flex flex-col items-center justify-center py-6 px-4 rounded-lg bg-[#141414] border border-[#242424] text-center">
                          <Boxes className="w-7 h-7 text-neutral-600 mb-1.5" />
                          <span className="text-xs font-medium text-neutral-300">
                            Keine Container mit diesem Netzwerk verbunden
                          </span>
                          <span className="text-[11px] text-neutral-500 mt-0.5">
                            Container können über Docker Compose oder `docker run --net` angehängt werden.
                          </span>
                        </div>
                      ) : (
                        <div className="rounded-lg bg-[#141414] border border-[#242424] overflow-x-auto">
                          <table className="w-full text-left text-xs">
                            <thead>
                              <tr className="border-b border-[#222222] bg-[#121212]/80 text-[10px] uppercase tracking-wider text-neutral-400 font-semibold font-sans">
                                <th className="py-2.5 px-3">Container</th>
                                <th className="py-2.5 px-3">Status</th>
                                <th className="py-2.5 px-3">IPv4-Adresse</th>
                                <th className="py-2.5 px-3">IPv6-Adresse</th>
                                <th className="py-2.5 px-3">MAC-Adresse</th>
                                <th className="py-2.5 px-3">Endpoint-ID</th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-[#1f1f1f] font-mono">
                              {containerEntries.map(([containerId, endpoint]) => {
                                const cleanName = endpoint.Name.replace(/^\//, '');
                                const shortId = containerId.slice(0, 12);
                                const isIPv4Copied = copiedKey === `${endpoint.EndpointID}-ip`;

                                return (
                                  <tr
                                    key={endpoint.EndpointID || containerId}
                                    className="hover:bg-white/[0.02] transition-colors"
                                  >
                                    {/* Container Name & ID */}
                                    <td className="py-2 px-3 font-sans">
                                      <div className="flex flex-col">
                                        <span className="text-xs font-medium text-white hover:text-blue-400 transition-colors">
                                          {cleanName}
                                        </span>
                                        <span className="text-[10px] font-mono text-neutral-500">
                                          ID: {shortId}
                                        </span>
                                      </div>
                                    </td>

                                    {/* Status Badge */}
                                    <td className="py-2 px-3 font-sans">
                                      <StatusBadge status={endpoint.State || 'running'} />
                                    </td>

                                    {/* IPv4 Address */}
                                    <td className="py-2 px-3">
                                      <div className="flex items-center gap-1.5">
                                        <span className="text-neutral-200">
                                          {endpoint.IPv4Address || '—'}
                                        </span>
                                        {endpoint.IPv4Address && (
                                          <button
                                            onClick={(e) =>
                                              copyToClipboard(
                                                endpoint.IPv4Address.split('/')[0],
                                                `${endpoint.EndpointID}-ip`,
                                                e
                                              )
                                            }
                                            className="text-neutral-500 hover:text-white transition-colors"
                                            title="IP kopieren"
                                          >
                                            {isIPv4Copied ? (
                                              <Check className="w-3 h-3 text-emerald-400" />
                                            ) : (
                                              <Copy className="w-3 h-3" />
                                            )}
                                          </button>
                                        )}
                                      </div>
                                    </td>

                                    {/* IPv6 Address */}
                                    <td className="py-2 px-3 text-neutral-400 text-[11px]">
                                      {endpoint.IPv6Address || '—'}
                                    </td>

                                    {/* MAC Address */}
                                    <td className="py-2 px-3 text-neutral-400 text-[11px]">
                                      {endpoint.MacAddress || '—'}
                                    </td>

                                    {/* Endpoint ID */}
                                    <td className="py-2 px-3 text-neutral-500 text-[11px]">
                                      {endpoint.EndpointID ? endpoint.EndpointID.slice(0, 12) : '—'}
                                    </td>
                                  </tr>
                                );
                              })}
                            </tbody>
                          </table>
                        </div>
                      )}
                    </div>

                    {/* Network Driver Options & Details */}
                    <div className="p-3 rounded-lg bg-[#141414] border border-[#242424] space-y-2">
                      <span className="text-[11px] font-semibold text-neutral-300 uppercase tracking-wider block font-sans">
                        Treiber-Optionen & Metadaten
                      </span>

                      <div className="flex items-center gap-2 flex-wrap text-xs">
                        {/* MTU */}
                        <span className="px-2 py-0.5 rounded bg-[#181818] border border-[#282828] text-neutral-300 font-mono text-[11px]">
                          MTU: {mtu || '1500 (Standard)'}
                        </span>

                        {/* ICC */}
                        <span
                          className={cn(
                            'inline-flex items-center gap-1 px-2 py-0.5 rounded border text-[11px] font-medium font-sans',
                            icc === 'false'
                              ? 'bg-amber-950/30 text-amber-400 border-amber-800/40'
                              : 'bg-[#181818] text-neutral-300 border-[#282828]'
                          )}
                        >
                          {icc === 'false' ? (
                            <>
                              <Lock className="w-3 h-3 text-amber-400" />
                              <span>ICC: Deaktiviert</span>
                            </>
                          ) : (
                            <>
                              <Unlock className="w-3 h-3 text-neutral-400" />
                              <span>ICC: Aktiviert</span>
                            </>
                          )}
                        </span>

                        {/* IP Masquerade */}
                        <span className="px-2 py-0.5 rounded bg-[#181818] border border-[#282828] text-neutral-300 font-mono text-[11px]">
                          NAT Masquerade: {masquerade === 'false' ? 'Aus' : 'Aktiv'}
                        </span>

                        {/* Attachable */}
                        {net.Attachable && (
                          <span className="px-2 py-0.5 rounded bg-blue-950/30 text-blue-400 border border-blue-800/40 text-[11px]">
                            Attachable
                          </span>
                        )}

                        {/* Compose Labels */}
                        {composeProject && (
                          <span className="px-2 py-0.5 rounded bg-[#181818] border border-[#282828] text-neutral-400 font-mono text-[11px]">
                            Compose: {composeProject}
                          </span>
                        )}
                        {composeNetwork && (
                          <span className="px-2 py-0.5 rounded bg-[#181818] border border-[#282828] text-neutral-400 font-mono text-[11px]">
                            Netzwerk: {composeNetwork}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Delete Confirmation Modal */}
      <ActionConfirmModal
        isOpen={deleteModal !== null}
        onClose={() => setDeleteModal(null)}
        onConfirm={handleDelete}
        title={`Netzwerk '${deleteModal?.Name}' löschen?`}
        description="Möchtest du dieses Docker-Netzwerk wirklich entfernen? Alle getrennten Container verlieren ihre Netzwerkverbindung auf diesem Adapter."
        isDestructive={true}
        confirmText="Löschen"
      />
    </div>
  );
};
