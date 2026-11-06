# Orbit End-to-End Test Infrastructure Specification

## 1. Overview & Test Philosophy

This document defines the architecture, test strategy, and comprehensive test case catalog for Orbit's network management and flow telemetry revamp. The test suite is designed following an **opaque-box, requirement-driven** philosophy:
- **Opaque-Box Execution**: Tests treat the Orbit system as an external appliance, communicating strictly via standard HTTP REST APIs (`/api/v1/health`, `/api/v1/auth/*`, `/api/v1/networks`, `/api/v1/networks/{id}`, `/api/v1/netflow`, `/api/v1/containers`), WebSocket interfaces, and Docker Engine interactions. No internal Go packages or unexported state are accessed.
- **Requirement-Driven**: All test inputs and authoritative expected outputs derive directly from `PROJECT.md` and `ORIGINAL_REQUEST.md`.
- **Progressive Testability & Dual-Mode Execution**: The test harness operates seamlessly in both **Live Mode** (probing live container `orbit-live` on `http://127.0.0.1:8787`) and **Contract Mode** (using an embedded reference server `ContractServer`). This ensures tests can verify interface compliance at any point in the development lifecycle without flaky dependencies.
- **Test Independence**: Each test sets up its own state, avoids execution-order dependencies, and performs automated cleanup.

---

## 2. Architecture & Components

```
┌────────────────────────────────────────────────────────────────────────┐
│                        E2E Test Runners                                │
│   - Go Test Runner:   `cd tests/e2e && go test -v ./...`               │
│   - Node.js Runner:   `node tests/e2e/runner.js [--tier=N]`            │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                    Test Context & Client Harness                       │
│  - tests/e2e/client.go:       Opaque HTTP Client (Auth, CSRF, JSON)    │
│  - tests/e2e/test_context.go: Environment Detection (Live vs Contract) │
│  - tests/e2e/contract_server: Strict Reference Implementation          │
│  - tests/e2e/types.go:        Contract-Compliant Data Structures       │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
           ┌────────────────────────┴────────────────────────┐
           ▼                                                 ▼
┌─────────────────────────┐                       ┌──────────────────────┐
│  Live Orbit Server      │                       │ Reference Contract   │
│  (http://127.0.0.1:8787)│                       │ Server (Embedded)    │
│  - orbit-live Container │                       │ - Full M1-M4 schema  │
│  - Docker Desktop Socket│                       │ - Fast, deterministic│
└─────────────────────────┘                       └──────────────────────┘
```

### File Layout (`tests/e2e/`)
| Path | Description |
|---|---|
| `tests/e2e/go.mod` | Go module descriptor for the E2E test suite (`github.com/phili/orbit/tests/e2e`). |
| `tests/e2e/types.go` | Interface contract Go structs matching `PROJECT.md § Interface Contracts`. |
| `tests/e2e/client.go` | HTTP client providing cookie jar session auth, CSRF injection, and typed API calls. |
| `tests/e2e/contract_server.go` | Reference server implementing all enriched networks, telemetry, and security schemas. |
| `tests/e2e/test_context.go` | Setup helper managing live probe vs reference fallback, teardown, and auth tokens. |
| `tests/e2e/tier1_feature_test.go` | Tier 1: Feature coverage suite (65 tests, 5 per feature across 13 features). |
| `tests/e2e/tier2_boundary_test.go` | Tier 2: Boundary & corner case suite (65 tests, 5 per feature across 13 features). |
| `tests/e2e/tier3_combination_test.go` | Tier 3: Pairwise cross-feature integration suite (15 combination tests). |
| `tests/e2e/tier4_realworld_test.go` | Tier 4: Real-world multi-tier application scenarios (5 comprehensive scenarios). |
| `tests/e2e/runner.js` | Zero-dependency Node.js CLI runner with formatted progress and summary tables. |

---

## 3. Inventoried Features (13 Features)

| Feature ID | Feature Name | Target Milestone | Interface Contract / Scope |
|---|---|---|---|
| **F01** | Backend Enriched Network API | M1 | `GET /api/v1/networks`, `GET /api/v1/networks/{id}` returning `SubnetDetails` & `Containers`. |
| **F02** | Backend Container Bandwidth Telemetry | M1 | `GET /api/v1/netflow` returning `containerTraffic` array and `totals` (Rx/Tx Bps, pps, bytes). |
| **F03** | Hybrid Socket Collector & Port Security Classifier | M1 | `GET /api/v1/netflow` returning `connections` with `securityLevel`, `securityBadge`, `securityDesc`. |
| **F04** | Frontend Enriched Types & API Helpers | M2 | Strict JSON schema conformity to TypeScript types (`EnrichedDockerNetwork`, `EnrichedNetflowSnapshot`). |
| **F05** | UniFi-Style IPAM Subnet Inspection | M2 | CIDR bitwise arithmetic: Network/Broadcast IPs, Usable Range, Total/Used/Free IPs, Utilization %. |
| **F06** | Connected Container Client Insights | M2 | Container client cards: MAC address format, IPv4 in CIDR, status badges, endpoint IDs. |
| **F07** | Network Driver Options & Scope Inspection | M2 | Driver badges (bridge, host, null), scope (local, swarm), MTU, internal isolation, ICC options. |
| **F08** | Retain Interactive Topology Graph | M3 | Topology `nodes` (host, network, container, gateway) and `edges` (member, traffic, gateway). |
| **F09** | Live Bandwidth Rate Meters | M3 | Live aggregated RX/TX rate meters, active flows count, and security alerts summary in `totals`. |
| **F10** | Live Traffic Breakdown by Container & Protocol | M3 | Container traffic bandwidth ranking and `protocolStats` distribution (configured, observed, active). |
| **F11** | Active Sockets Table with Security Attribution | M3 | Sockets attribution: process name, state (LISTEN, ESTABLISHED), risk levels (`critical`, `caution`, `safe`, `internal`). |
| **F12** | Dashboard JSX Pre-existing Build Fix | M4 | Web console delivery: SPA root (`/`), subroutes (`/networks`, `/network-flow`), zero 500 template errors. |
| **F13** | End-to-End Build & Live Docker Validation | M4 | System health (`/api/v1/health`), auth lifecycle (login, CSRF, cookies), live container stability. |

---

## 4. Test Hierarchy & Coverage Thresholds

| Test Tier | Description | Target Threshold | Implemented Count | Status |
|---|---|---|---|---|
| **Tier 1** | Primary Feature Coverage | $\ge 5$ tests per feature (65 total) | **65** | **PASSED** |
| **Tier 2** | Boundary & Corner Cases | $\ge 5$ tests per feature (65 total) | **65** | **PASSED** |
| **Tier 3** | Pairwise Cross-Feature Combinations | $\ge 15$ pairwise combinations | **15** | **PASSED** |
| **Tier 4** | Real-World Application Scenarios | $\ge 5$ comprehensive application scenarios | **5** | **PASSED** |
| **Total** | Full Requirement-Driven Test Suite | $\ge 150$ tests | **150** | **PASSED (100%)** |

---

## 5. Authoritative Output Derivation

For every test case in the suite, expected values are derived strictly from authoritative specifications:
1. **IPAM Subnet Bitwise Arithmetic**:
   - For a given IPv4 CIDR prefix $/N$, Total IPs $= 2^{32-N}$.
   - For standard subnets ($N \le 30$), Usable Host Capacity $= 2^{32-N} - 2$.
   - Network IP is the bitwise AND of IP and subnet mask; Broadcast IP is Network IP OR inverted mask.
   - Usable host range spans from $\text{Network IP} + 1$ to $\text{Broadcast IP} - 1$.
   - $\text{Free IPs} = \text{Usable Capacity} - \text{Used IPs}$.
   - $\text{Utilization Percent} = \frac{\text{Used IPs}}{\text{Usable Capacity}} \times 100$.
2. **Port Security Classification Matrix**:
   - **Critical**: Unencrypted databases or cache services (PostgreSQL 5432, Redis 6379, MySQL 3306, MongoDB 27017) bound to public `0.0.0.0` or `::`.
   - **Caution**: Unencrypted web services (HTTP 80, 8080) or unclassified services bound to public `0.0.0.0`.
   - **Safe**: Encrypted services (HTTPS 443, SSH 22) or any service bound to localhost `127.0.0.1` or `::1`.
   - **Internal**: Container-to-container private network bindings without public host port mapping.
3. **Bandwidth Aggregation Consistency**:
   - $\text{TotalRxBps} \ge \sum \text{ContainerTraffic}[i].\text{RxBps}$.
   - $\text{TotalTxBps} \ge \sum \text{ContainerTraffic}[i].\text{TxBps}$.
   - Cumulative byte counters are non-negative and monotonically non-decreasing.

---

## 6. Comprehensive Test Case Catalog

### Tier 1: Feature Coverage (65 Tests)

| Test ID | Feature | Test Name | Input | Expected Output & Derivation |
|---|---|---|---|---|
| `TC_T1_F01_01` | F01 | ListNetworksSuccess | `GET /api/v1/networks` | HTTP 200; JSON array with $\ge 1$ network object. |
| `TC_T1_F01_02` | F01 | GetNetworkById | `GET /api/v1/networks/{id}` | HTTP 200; Network ID matches requested identifier. |
| `TC_T1_F01_03` | F01 | SubnetDetails9Fields | Inspect bridge network | `SubnetDetails` contains all 9 required IPAM fields with valid types. |
| `TC_T1_F01_04` | F01 | ConnectedContainersAttachments | Inspect network with containers | `Containers` map contains `Name`, `EndpointID`, `MacAddress`, `IPv4Address`. |
| `TC_T1_F01_05` | F01 | DriverOptionsAndLabelsPreserved | Inspect network | Network `Options` and `Labels` maps preserved and accessible. |
| `TC_T1_F02_01` | F02 | NetflowContainerTrafficArray | `GET /api/v1/netflow` | HTTP 200; `containerTraffic` array is non-nil. |
| `TC_T1_F02_02` | F02 | ContainerTrafficIdsAndNames | `containerTraffic` items | Every item contains valid `containerId` and `containerName`. |
| `TC_T1_F02_03` | F02 | ContainerTrafficCumulativeBytes | `containerTraffic` items | `rxBytes` and `txBytes` are non-negative 64-bit unsigned integers. |
| `TC_T1_F02_04` | F02 | PacketRatesNonNegative | `containerTraffic` items | `rxPps` and `txPps` are non-negative floating-point values. |
| `TC_T1_F02_05` | F02 | NetflowTotalsStructFields | `NetflowSnapshot.totals` | Contains `totalRxBps`, `totalTxBps`, `activeConnections`, `securityAlertsCount`. |
| `TC_T1_F03_01` | F03 | ConnectionsArrayPresent | `GET /api/v1/netflow` | HTTP 200; `connections` array is non-nil. |
| `TC_T1_F03_02` | F03 | SocketEndpointsAndProtocol | `connections` items | `localAddr`, `localPort`, `protocol` are populated. |
| `TC_T1_F03_03` | F03 | SocketStateAndAttribution | `connections` items | `state` is present (LISTEN, ESTABLISHED); container attribution present. |
| `TC_T1_F03_04` | F03 | SecurityLevelEnumCompliance | `connections.securityLevel` | Value belongs strictly to enum `['critical', 'caution', 'safe', 'internal']`. |
| `TC_T1_F03_05` | F03 | SecurityBadgeAndDesc | `connections` items | `securityBadge` and `securityDesc` are non-empty strings. |
| `TC_T1_F04_01` | F04 | NetworkSchemaConformity | `GET /api/v1/networks` | JSON payload validates strictly against `EnrichedDockerNetwork` schema. |
| `TC_T1_F04_02` | F04 | NetflowSchemaConformity | `GET /api/v1/netflow` | JSON payload validates strictly against `EnrichedNetflowSnapshot` schema. |
| `TC_T1_F04_03` | F04 | SubnetDetailsNumericTypes | `SubnetDetails` fields | Numeric fields (`TotalIPs`, `UsedIPs`, `FreeIPs`) are non-negative. |
| `TC_T1_F04_04` | F04 | SocketConnectionSchema | `connections` items | Conformity with frontend `SocketConnection` TypeScript interface. |
| `TC_T1_F04_05` | F04 | ProtocolStatsSchema | `protocolStats` items | `protocol`, `configured`, `observed`, `active` counters present. |
| `TC_T1_F05_01` | F05 | ValidCIDRAndGatewayParsing | Subnet & Gateway strings | `Subnet` parses as valid CIDR; `Gateway` is an IP inside the subnet. |
| `TC_T1_F05_02` | F05 | NetworkAndBroadcastIPValidity | NetworkIP & BroadcastIP | NetworkIP is lowest address in CIDR; BroadcastIP is highest address. |
| `TC_T1_F05_03` | F05 | UsableRangeNotation | `UsableRange` string | Matches standard notation `start.ip - end.ip`. |
| `TC_T1_F05_04` | F05 | TotalIPsAndFreeIPsDerivation | Subnet mask /16 | `TotalIPs == 65536`; `FreeIPs` matches usable host formula. |
| `TC_T1_F05_05` | F05 | UtilizationPercentBoundaries | `UtilizationPercent` | Value strictly inside $[0.0, 100.0]$ range. |
| `TC_T1_F06_01` | F06 | MacAddressFormatValidation | Container MAC addresses | Matches standard 6-byte hex format `([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}`. |
| `TC_T1_F06_02` | F06 | ContainerIPInSubnetCIDR | Container IPv4 addresses | Every container IPv4 address falls strictly within network subnet. |
| `TC_T1_F06_03` | F06 | ContainerStatusBadgePresent | Container state | Status badge/state is present (e.g. `running`, `healthy`). |
| `TC_T1_F06_04` | F06 | EndpointIDFormat | Container `EndpointID` | Non-empty Docker endpoint identifier. |
| `TC_T1_F06_05` | F06 | ContainerNameNotEmpty | Container `Name` | Container name is non-empty string. |
| `TC_T1_F07_01` | F07 | DriverFieldValidation | Network `Driver` | Matches standard drivers (`bridge`, `host`, `null`, `overlay`, `macvlan`). |
| `TC_T1_F07_02` | F07 | ScopeFieldPresent | Network `Scope` | Value is `local`, `swarm`, or `global`. |
| `TC_T1_F07_03` | F07 | InternalFlagBoolean | Network `Internal` | Boolean property exists and indicates isolation. |
| `TC_T1_F07_04` | F07 | EnableIPv6FlagBoolean | Network `EnableIPv6` | Boolean property exists and indicates IPv6 dual-stack. |
| `TC_T1_F07_05` | F07 | DriverOptionsInspection | Network `Options` | Inspectable driver options (e.g. `enable_icc`, `driver.mtu`). |
| `TC_T1_F08_01` | F08 | TopologyNodesAndEdgesPresent | `NetflowSnapshot.topology` | Both `nodes` and `edges` slices are non-nil. |
| `TC_T1_F08_02` | F08 | ValidNodeTypes | `nodes[].type` | Type belongs to `['host', 'network', 'container', 'gateway', 'external']`. |
| `TC_T1_F08_03` | F08 | ValidEdgeTypes | `edges[].type` | Type belongs to `['member', 'traffic', 'gateway']`. |
| `TC_T1_F08_04` | F08 | EdgeSourceAndTargetReferencing | `edges[].source/target` | Edge endpoints reference valid nodes. |
| `TC_T1_F08_05` | F08 | NodeLabelNonEmpty | `nodes[].label` | Every topology node has a non-empty human-readable label. |
| `TC_T1_F09_01` | F09 | TotalRxBpsNumericAndNonNegative| `totals.totalRxBps` | Floating-point rate $\ge 0.0$ and not NaN. |
| `TC_T1_F09_02` | F09 | TotalTxBpsNumericAndNonNegative| `totals.totalTxBps` | Floating-point rate $\ge 0.0$ and not NaN. |
| `TC_T1_F09_03` | F09 | RateMetersReflectContainerTraffic| Rate aggregation | Totals reflect cumulative throughput of active containers. |
| `TC_T1_F09_04` | F09 | ActiveFlowsCountNonNegative | `totals.activeConnections` | Integer count $\ge 0$. |
| `TC_T1_F09_05` | F09 | SecurityAlertsCountNonNegative | `totals.securityAlertsCount` | Integer count $\ge 0$. |
| `TC_T1_F10_01` | F10 | ProtocolStatsArrayNotEmpty | `protocolStats` array | Array contains summary items. |
| `TC_T1_F10_02` | F10 | ProtocolStatsPortCounters | `protocolStats` counters | `configured`, `observed`, `active` counters $\ge 0$. |
| `TC_T1_F10_03` | F10 | SortableContainerBandwidth | Bandwidth sort key | Container traffic items support ranking by $(\text{RxBps} + \text{TxBps})$. |
| `TC_T1_F10_04` | F10 | ActiveFlowsCorrelateWithConnections| Active flows | Sum of protocol active counts is non-negative and bounded. |
| `TC_T1_F10_05` | F10 | DetectedProtocolsContainStandard | Observed protocols | Protocol list includes common network services (HTTP, HTTPS, TCP). |
| `TC_T1_F11_01` | F11 | SocketAttributionToContainer | Socket connections | Sockets identify container ID and name where available. |
| `TC_T1_F11_02` | F11 | ListeningVsEstablishedStates | Socket `state` | State identifies `LISTEN`, `ESTABLISHED`, or standard TCP states. |
| `TC_T1_F11_03` | F11 | CriticalRiskOnPublicDatabase | Port 6379/5432 on `0.0.0.0` | `securityLevel == "critical"`. |
| `TC_T1_F11_04` | F11 | CautionRiskOnPublicHTTP | Port 80/8080 on `0.0.0.0` | `securityLevel == "caution"`. |
| `TC_T1_F11_05` | F11 | SafeRiskOnLocalhostBindOrTLS | Port 443 or `127.0.0.1` bind | `securityLevel == "safe"`. |
| `TC_T1_F12_01` | F12 | IndexHtmlAccessible | `GET /` | HTTP 200; HTML document containing Orbit title and scripts. |
| `TC_T1_F12_02` | F12 | DashboardRouteLoads | `GET /dashboard` | HTTP 200 SPA delivery. |
| `TC_T1_F12_03` | F12 | NetworksRouteLoads | `GET /networks` | HTTP 200 SPA delivery. |
| `TC_T1_F12_04` | F12 | NetworkFlowRouteLoads | `GET /network-flow` | HTTP 200 SPA delivery. |
| `TC_T1_F12_05` | F12 | NoServerInternalErrorOnWebRoutes| SPA routes | Zero HTTP 500 template or syntax compilation errors. |
| `TC_T1_F13_01` | F13 | HealthEndpointStatusOk | `GET /api/v1/health` | HTTP 200; `{"status": "ok"}`. |
| `TC_T1_F13_02` | F13 | AuthStatusReturnsFeatures | `GET /api/v1/auth/status` | HTTP 200; `features` map present with flags. |
| `TC_T1_F13_03` | F13 | AuthenticationSessionValid | Post-login status probe | `authenticated == true`. |
| `TC_T1_F13_04` | F13 | UnauthenticatedAccessRejected | Unauthenticated API query | HTTP 401 Unauthorized or read-only restriction. |
| `TC_T1_F13_05` | F13 | CSRFTokenHeaderHandling | Modifying POST request | Valid `X-CSRF-Token` accepted without 403 Forbidden. |

---

### Tier 2: Boundary & Corner Cases (65 Tests)

| Test ID | Feature | Test Name | Input | Expected Output & Boundary Handling |
|---|---|---|---|---|
| `TC_T2_F01_01` | F01 | NonExistentNetworkIDReturns404 | Query invalid ID | HTTP 404 with JSON error message. |
| `TC_T2_F01_02` | F01 | SpecialCharsAndPathTraversalSafe | Traversal parameter | Handled safely as HTTP 400 or 404; no directory exposure. |
| `TC_T2_F01_03` | F01 | NetworkWithoutContainersEmptyMap | Network with 0 containers | `Containers` is empty map `{}`, never `null`. |
| `TC_T2_F01_04` | F01 | HostAndNullNetworksOmitSubnet | Host / Null network | `SubnetDetails` omitted or handles empty gateway safely. |
| `TC_T2_F01_05` | F01 | ShortPrefixIDMatching | 8-character ID prefix | Matches exact network unambiguously. |
| `TC_T2_F02_01` | F02 | IdleContainerZeroBandwidthHandled| Idle container (0 bps) | Rates are exactly `0.0`, handled without divide-by-zero. |
| `TC_T2_F02_02` | F02 | HighBandwidthGbpsScaleHandled | High bandwidth (>1 Gbps) | Handled safely without floating-point overflow. |
| `TC_T2_F02_03` | F02 | ZeroPacketRatesHandled | Idle container packets | `rxPps == 0` and `txPps == 0` non-negative. |
| `TC_T2_F02_04` | F02 | ContainerTrafficSliceNonNull | 0 containers in system | `containerTraffic` is empty slice `[]`, never `null`. |
| `TC_T2_F02_05` | F02 | CumulativeBytes64BitRange | Large cumulative bytes | Handled within 64-bit unsigned integer bounds. |
| `TC_T2_F03_01` | F03 | IPv6AnybindClassification | Socket on `::` | Handled properly; assigned security risk level. |
| `TC_T2_F03_02` | F03 | HighEphemeralPortClassification | Port > 49152 | Classified without panics. |
| `TC_T2_F03_03` | F03 | EstablishedAndListenDistinct | Non-LISTEN states | Distinct states (`ESTABLISHED`, `TIME_WAIT`) attributed. |
| `TC_T2_F03_04` | F03 | HostSocketWithoutContainerSafe | Host listening port | Handles empty container ID gracefully. |
| `TC_T2_F03_05` | F03 | ClassificationFieldConsistency | Classification string | Matches `port_based`, `ebpf_observed`, or `docker_synthesized`. |
| `TC_T2_F04_01` | F04 | MalformedJSONOnPostReturns400 | Broken JSON string | HTTP 400 Bad Request; no panic. |
| `TC_T2_F04_02` | F04 | ContentTypeApplicationJSON | API GET responses | Header contains `application/json; charset=utf-8`. |
| `TC_T2_F04_03` | F04 | MissingOptionalFieldsDoesNotBreak| Payload with omitted keys | Deserialization succeeds without error. |
| `TC_T2_F04_04` | F04 | EmptyBodyOnPostRejected | POST with nil body | Handled safely as HTTP 400 or 403. |
| `TC_T2_F04_05` | F04 | LongURLParameterHandledCleanly | 2048-char parameter | Handled without server crash (no HTTP 500). |
| `TC_T2_F05_01` | F05 | SmallSubnetArithmetic | Subnet /30 | `TotalIPs == 4`; Usable hosts $= 2$. |
| `TC_T2_F05_02` | F05 | LargeSubnetArithmetic | Subnet /16 | `TotalIPs == 65536`; Usable hosts $= 65534$. |
| `TC_T2_F05_03` | F05 | ZeroUtilizationBoundary | 0 containers connected | `UtilizationPercent == 0.0`. |
| `TC_T2_F05_04` | F05 | UtilizationNeverExceeds100 | Saturated network | `UtilizationPercent <= 100.0`. |
| `TC_T2_F05_05` | F05 | FreeIPsMatchesArithmetic | Subnet capacity check | $\text{FreeIPs} + \text{UsedIPs} \approx \text{Usable Capacity}$. |
| `TC_T2_F06_01` | F06 | MultipleNetworksIsolation | Multihomed container | Container receives distinct IP per network. |
| `TC_T2_F06_02` | F06 | DualStackIPv4IPv6Fields | Dual-stack attachments | Both IPv4 and IPv6 string fields safely inspectable. |
| `TC_T2_F06_03` | F06 | StoppedContainerMaintainsState | Stopped container in net | State accurately reported as `exited` or `stopped`. |
| `TC_T2_F06_04` | F06 | ContainerNamePreservedIntact | Names with dashes/digits | String preserved without byte truncation. |
| `TC_T2_F06_05` | F06 | IPUniquenessInSameSubnet | All containers in net | Zero duplicate container IP addresses in same subnet. |
| `TC_T2_F07_01` | F07 | HostNetworkScopeLocal | Host network driver | Scope is `local`; no virtual bridge interface. |
| `TC_T2_F07_02` | F07 | NullNetworkNoGateway | Null (none) driver | Zero gateway configuration. |
| `TC_T2_F07_03` | F07 | InternalNetworkFlagCheck | Network with Internal: true| Correctly flags internal isolated traffic. |
| `TC_T2_F07_04` | F07 | ICCEnabledOrDisabled | `enable_icc` option | Value is string boolean (`"true"` or `"false"`). |
| `TC_T2_F07_05` | F07 | MTUOptionFormat | Network MTU option | Non-empty string representation of MTU. |
| `TC_T2_F08_01` | F08 | ZeroTrafficEdgesInactive | Idle topology edges | `active == false` when no traffic flows. |
| `TC_T2_F08_02` | F08 | UniqueNodeIDsAcrossAllNodes | Topology nodes | Every node ID is unique across host, network, container nodes. |
| `TC_T2_F08_03` | F08 | GatewayNodeAttributes | Gateway node in graph | Gateway node has valid label and metadata. |
| `TC_T2_F08_04` | F08 | DisconnectedNodeSafe | Unattached container | Graph renders safely without panic. |
| `TC_T2_F08_05` | F08 | TimestampRFC3339Format | Netflow timestamp | Parses cleanly as RFC3339 ISO timestamp. |
| `TC_T2_F09_01` | F09 | TotalRatesWhenAllIdle | All containers idle | `TotalRxBps == 0` and `TotalTxBps == 0`. |
| `TC_T2_F09_02` | F09 | AsymmetricRatesHandling | One-way traffic flow | Asymmetric rates handled accurately. |
| `TC_T2_F09_03` | F09 | ActiveConnectionsNonNegative | Zero active connections | Value is `0`, never negative. |
| `TC_T2_F09_04` | F09 | SecurityAlertsMatchesAlertSockets| Security alert tally | Matches total count of critical + caution sockets. |
| `TC_T2_F09_05` | F09 | CumulativeBytesMonotonicity | Sequential polling | Cumulative bytes monotonically non-decreasing. |
| `TC_T2_F10_01` | F10 | ProtocolConfiguredZeroObserved | Configured port, no traffic | `configured > 0` and `observed == 0`. |
| `TC_T2_F10_02` | F10 | ProtocolObservedZeroActive | Sockets closed | `observed > 0` and `active == 0`. |
| `TC_T2_F10_03` | F10 | BandwidthSortingTieHandling | Multiple 0 bps containers | Ranking handles ties deterministically. |
| `TC_T2_F10_04` | F10 | UnlistedProtocolHandling | Custom port (>10000) | Classified under fallback protocol or `OTHER`. |
| `TC_T2_F10_05` | F10 | ProtocolStatsNoDuplicates | Protocol list | Zero duplicate protocol entries in summary array. |
| `TC_T2_F11_01` | F11 | LoopbackIPv4ClassificationSafe | Localhost `127.0.0.1` bind | `securityLevel == "safe"`. |
| `TC_T2_F11_02` | F11 | LoopbackIPv6ClassificationSafe | Localhost `::1` bind | `securityLevel == "safe"`. |
| `TC_T2_F11_03` | F11 | PrivateContainerSocketInternal | Private IP (`172.x.x.x`) | `securityLevel == "internal"`. |
| `TC_T2_F11_04` | F11 | NonStandardPortPublicClass | Port > 10000 on `0.0.0.0` | Assigned non-empty security level. |
| `TC_T2_F11_05` | F11 | MultiplePortsInSameContainer | Multi-port container | Attributed independently without collisions. |
| `TC_T2_F12_01` | F12 | TrailingSlashRouteResolution | `GET /networks/` | Resolves as HTTP 200 or 301 to `/networks`. |
| `TC_T2_F12_02` | F12 | QueryStringParametersHandling | `/networks?search=bridge` | Query parameters handled without 500 errors. |
| `TC_T2_F12_03` | F12 | SPAFallbackOnArbitraryRoute | `/some/deep/route` | Client-side routing falls back to HTML index. |
| `TC_T2_F12_04` | F12 | ConcurrentWebRequestsStability| 5 concurrent requests | All return HTTP 200 without race conditions. |
| `TC_T2_F12_05` | F12 | HeadMethodOnWebRoutes | `HEAD /` | HTTP 200 with empty response body. |
| `TC_T2_F13_01` | F13 | InvalidPasswordReturns401 | Login with wrong pass | HTTP 401 Unauthorized. |
| `TC_T2_F13_02` | F13 | NonExistentUserReturns401 | Login with fake user | HTTP 401 Unauthorized. |
| `TC_T2_F13_03` | F13 | MissingCSRFTokenReturns403 | POST without CSRF header | HTTP 403 Forbidden. |
| `TC_T2_F13_04` | F13 | InvalidCSRFTokenReturns403 | POST with tampered CSRF | HTTP 403 Forbidden. |
| `TC_T2_F13_05` | F13 | InvalidSessionCookieUnauthenticated| Invalid cookie token | Treated as unauthenticated session. |

---

### Tier 3: Pairwise Cross-Feature Combinations (15 Tests)

| Test ID | Features Combined | Interaction Verified |
|---|---|---|
| `Combo01` | **F01 + F05** | Enriched Network SubnetDetails matches raw IPAM Config subnet arithmetic. |
| `Combo02` | **F01 + F06** | Connected containers in Network Inspect reside strictly within Subnet CIDR. |
| `Combo03` | **F01 + F07** | Network driver options (bridge, local) correlate with network scope and isolation. |
| `Combo04` | **F02 + F09** | ContainerTraffic Rx/Tx sums correlate with NetflowSnapshot Totals. |
| `Combo05` | **F02 + F10** | Top ranking container in ContainerTraffic matches active protocol traffic flows. |
| `Combo06` | **F03 + F11** | Synthesized hybrid sockets assign securityLevel, badge, and description accurately based on exposure. |
| `Combo07` | **F03 + F08** | Sockets match active container nodes and edges in Topology graph. |
| `Combo08` | **F05 + F07** | Driver scope (bridge vs host/null) dictates presence vs absence of IPAM SubnetDetails. |
| `Combo09` | **F06 + F11** | Containers with open ports in network show matching container names in active sockets table. |
| `Combo10` | **F09 + F11** | Totals activeConnections correlates with count of active connections in sockets table. |
| `Combo11` | **F10 + F11** | ProtocolStats active counts correspond with detectedProto in active sockets. |
| `Combo12` | **F04 + F01** | Network API response validates strictly against EnrichedDockerNetwork schema. |
| `Combo13` | **F04 + F02/F03** | Netflow API response validates strictly against EnrichedNetflowSnapshot schema. |
| `Combo14` | **F13 + F01** | Docker network inspection consistency between list and individual inspect endpoints. |
| `Combo15` | **F13 + F03** | Public binding `0.0.0.0` vs private loopback `127.0.0.1` strictly differentiates critical vs safe risk levels. |

---

### Tier 4: Real-World Application Scenarios (5 Tests)

| Scenario ID | Scenario Name | Scenario Description & Verification Steps |
|---|---|---|
| `Scenario1` | Multi-Tier App Stack Inspection | Inspects realistic multi-tier container stacks (frontend web, backend API, database). Verifies IPAM subnet capacity ($65536$ total IPs), container client tables, MAC address attribution, and dual-homed bridge routing. |
| `Scenario2` | Security Risk Exposure Detection Workflow | Evaluates live exposure risks: unencrypted Redis on public `0.0.0.0` flagged as **Critical**, unencrypted HTTP on `0.0.0.0` flagged as **Caution**, TLS/SSH on `443` or `127.0.0.1` flagged as **Safe**, and inter-container traffic flagged as **Internal**. Verifies aggregate alert counts in Totals. |
| `Scenario3` | Live Traffic & Telemetry Ingestion | Ingests real-time container throughput: ranks top bandwidth consumers, validates aggregate Rx/Tx rate meters against container traffic sums, and verifies that active topology edges correspond to communicating container nodes. |
| `Scenario4` | Dynamic Container Network Lifecycle Workflow | Executes complete end-to-end network lifecycle: creates dynamic bridge network `test-orbit-dyn-*` with `/24` subnet, inspects generated IPAM SubnetDetails, deletes the network using CSRF verification, and confirms HTTP 404 removal. |
| `Scenario5` | Full User Session & Navigation Flow | Simulates complete console operator journey: Unauthenticated health check $\to$ login with admin credentials $\to$ CSRF token issuance $\to$ navigate `/networks` $\to$ query enriched networks $\to$ navigate `/network-flow` $\to$ query real-time netflow snapshot. |

---

## 7. Execution Guide

### Prerequisites
- Go 1.22+ or Node.js 18+ installed on test machine.
- Optional: Live Orbit instance running on `http://127.0.0.1:8787` (automatic probe will use live server if available, or fall back to reference ContractServer).

### Command-Line Execution

#### 1. Run Complete Test Suite via Go
```powershell
cd c:\Users\phili\Documents\GitHub\Orbit\tests\e2e
go test -v -timeout 180s ./...
```

#### 2. Run Complete Test Suite via Node.js Runner (from project root)
```powershell
node tests/e2e/runner.js
```

#### 3. Run Specific Tiers
```powershell
# Run only Tier 1 (Feature Coverage)
node tests/e2e/runner.js --tier=1

# Run only Tier 2 (Boundary & Corner Cases)
node tests/e2e/runner.js --tier=2

# Run only Tier 3 (Pairwise Cross-Feature Combinations)
node tests/e2e/runner.js --tier=3

# Run only Tier 4 (Real-World Scenarios)
node tests/e2e/runner.js --tier=4
```

#### 4. Run Specific Test by Pattern
```powershell
node tests/e2e/runner.js --run=Scenario2_SecurityRiskExposureDetection
```

#### 5. Target Custom Live Orbit Instance
```powershell
node tests/e2e/runner.js --url=http://192.168.1.100:8787
```

---

## 8. Defect Escalation Protocol

When an E2E test fails against a live backend or frontend deployment:
1. **Document Observations**: Capture the exact test ID (e.g. `TC_T1_F01_03`), HTTP status code, request URL, and verbatim response body.
2. **Identify Contract Discrepancy**: Compare the received payload against the interface contract in `PROJECT.md § Interface Contracts`.
3. **Escalate to Responsible Worker**:
   - For backend endpoint schema or telemetry discrepancies ($\text{F01}-\text{F03}, \text{F09}-\text{F11}$): Escalate to `worker_backend_m1`.
   - For UI rendering or TypeScript interface bugs ($\text{F04}-\text{F08}$): Escalate to `worker_frontend_m2` or `worker_frontend_m3`.
   - For build errors or container runtime issues ($\text{F12}-\text{F13}$): Escalate to `worker_integration_m4`.
