# Forensic Audit Report: Milestone M1 (Backend Telemetry & Enriched Endpoints)

**Work Product:** Milestone M1 Backend Changes (`backend/internal/api/`, `backend/internal/netflow/`)  
**Integrity Mode:** `development` (verified directly from `ORIGINAL_REQUEST.md:8`)  
**Profile:** General Project  
**Verdict:** **CLEAN**  

---

## 1. Observation

Direct empirical observations from source analysis, forensic scans, test executions, and compilation:

1. **Pre-populated Artifact Check:**
   - Command: `Get-ChildItem -Path backend -Recurse -Include *.log,*result*,*output*,*fixture* -File`
   - Result: 0 files found. No pre-populated test fixtures, results, or logs exist in the repository.

2. **Hardcoded Test Result & Facade Inspection:**
   - Inspected `calculateSubnetDetails` in `backend/internal/api/handlers.go:412-517`:
     - Parses input CIDR via standard library `net.ParseCIDR(subnetStr)`.
     - Validates IPv4: `ip4 := ipnet.IP.To4()`.
     - Calculates host bits dynamically: `ones, bits := ipnet.Mask.Size(); hostBits := 32 - ones`.
     - Genuine broadcast bitwise math:
       ```go
       broadcast := make(net.IP, 4)
       for i := 0; i < 4; i++ {
           broadcast[i] = ip4[i] | ^ipnet.Mask[i]
       }
       ```
     - Computes first/last usable IP via big-endian byte encoding:
       ```go
       binary.BigEndian.PutUint32(firstUsable, netUint+1)
       binary.BigEndian.PutUint32(lastUsable, bcastUint-1)
       ```
     - Dynamic total IP calculation: `totalIPs := 1 << hostBits`.
     - Handles runtime Docker status (`status.IPAM.Subnets`) when present, falling back to subnet membership verification (`ipnet.Contains(parsed)`) for attached containers and gateway.
     - Handles edge cases: `/32` (1 host), `/31` (point-to-point RFC 3021), `/30`, `/16`, `/8`, invalid CIDR, empty string, and IPv6.
     - **Finding:** No hardcoded return values, lookup tables, or test-name-specific conditionals.

3. **Port Security Classifier Inspection:**
   - Inspected `AssessPortSecurity` in `backend/internal/netflow/protocols.go:56-113`:
     - Evaluates localhost bindings (`127.0.0.1`, `::1`, `localhost`, `127.*`) -> `safe` ("Safe").
     - Evaluates private networks (`parsedIP.IsPrivate() || parsedIP.IsLinkLocalUnicast()`) -> `internal` ("Internal").
     - Evaluates encrypted services (ports 443, 8443, 22, 2376 or detectedProto HTTPS, SSH, Docker TLS) on public IP -> `safe` ("Safe").
     - Evaluates unencrypted critical databases/caches (ports 6379, 27017, 5432, 3306, 11211, 9200, 2375 or detectedProto Redis, MongoDB, PostgreSQL, MySQL, Memcached, Elasticsearch, Docker Daemon) on public IP -> `critical` ("Critical Risk").
     - Evaluates unencrypted HTTP (80, 8080, 8000, 8888) or unclassified ports on public IP -> `caution` ("Caution").
     - **Finding:** Authentic risk-tier heuristics based on IP binding, protocol encryption, and port metadata. No hardcoded fixtures.

4. **Container Socket Synthesis & Streaming Telemetry Inspection:**
   - Inspected `synthesizeContainerSockets` and `extractProcessName` in `backend/internal/netflow/socket_reader.go:89-253`:
     - Genuinely inspects Docker container structures (`c.State`, `c.Ports`, `c.ExposedPorts`, `c.Networks`).
     - Extracts command binary name via `filepath.Base(strings.Trim(fields[0], "\"'"))`, with image name fallback.
     - Attributes listening sockets to container ID, name, process name, and runs `AssessPortSecurity`.
   - Inspected `BuildSnapshot` in `backend/internal/netflow/builder.go:32-276`:
     - Reads live container metrics from `b.streamer.GetAllLatest()`.
     - Extracts `NetRxBps`, `NetTxBps`, `NetRxBytes`, `NetTxBytes`, `NetRxPps`, `NetTxPps`.
     - Sorts traffic descending by bandwidth: `sort.Slice(traffic, ...)`.
     - Aggregates `Totals` (`TotalRxBps`, `TotalTxBps`, `ActiveConnections`, `SecurityAlertsCount`).
     - Enforces non-nil initialized slices (`[]ContainerTrafficMetric{}`, `[]SocketConnection{}`, `[]ProtocolSummary{}`) to prevent JSON `null` values.
     - **Finding:** Authentic data aggregation directly from Docker inventory and stats streamer.

5. **Backdoor and Mock Bypass Audit:**
   - Searched for environment variable overrides, debug bypasses, auth skip headers, or backdoor logic in `backend/internal/api/` and `backend/internal/netflow/`.
   - Result: 0 backdoor or bypass mechanisms found.

6. **Static Checks and Test Builds:**
   - Ran `cd backend; go test ./...`:
     ```
     ?   	github.com/phili/orbit/cmd/orbit	[no test files]
     ?   	github.com/phili/orbit/internal/alerts	[no test files]
     ok  	github.com/phili/orbit/internal/api	0.196s
     ok  	github.com/phili/orbit/internal/auth	(cached)
     ?   	github.com/phili/orbit/internal/config	[no test files]
     ok  	github.com/phili/orbit/internal/docker	(cached)
     ok  	github.com/phili/orbit/internal/gpu	(cached)
     ok  	github.com/phili/orbit/internal/host	(cached)
     ok  	github.com/phili/orbit/internal/metrics	(cached)
     ?   	github.com/phili/orbit/internal/model	[no test files]
     ok  	github.com/phili/orbit/internal/netflow	(cached)
     ?   	github.com/phili/orbit/internal/realtime	[no test files]
     ?   	github.com/phili/orbit/internal/settings	[no test files]
     ?   	github.com/phili/orbit/internal/store	[no test files]
     ?   	github.com/phili/orbit/web	[no test files]
     ```
     Exit code: 0. All unit tests across all packages passed.
   - Ran `cd backend; go build ./...`:
     Exit code: 0. Zero compiler errors, zero warnings.

---

## 2. Logic Chain

1. **Integrity Mode Conformance:**
   - Under `development` integrity mode (specified in `ORIGINAL_REQUEST.md`), prohibited patterns are:
     - Hardcoded test results / strings
     - Facade / dummy implementations
     - Fabricated verification outputs
   - Source code analysis confirmed that `calculateSubnetDetails`, `AssessPortSecurity`, `synthesizeContainerSockets`, and `BuildSnapshot` perform genuine mathematical bitwise calculations, network inspections, and telemetry aggregation.
   - Pre-populated artifact detection found zero artificial logs or fixture files.

2. **Mathematical and Security Soundness:**
   - Subnet arithmetic adheres to standard IPv4 bit operations (`1 << hostBits`, bitwise OR with bitwise NOT mask, big-endian uint32 boundary calculations).
   - Port security categorization correctly classifies loopback as safe, RFC 1918 as internal, encrypted protocols (TLS/SSH) as safe, and unencrypted databases/caches bound publicly (0.0.0.0/::) as critical risks.

3. **Acceptance Criteria Verification:**
   - Requirement R4 and Acceptance Criteria dictate: "Backend tests and builds pass cleanly (`go test ./...` and `go build ./...`)."
   - Independent execution confirmed both `go test ./...` and `go build ./...` succeed with exit code 0.

---

## 3. Caveats

- **Procfs Availability:** In Docker container environments without `/host/proc` (such as `orbit-live`), `SocketReader` activates its synthetic fallback from Docker container port inspects (`c.Ports`, `c.ExposedPorts`). In an idle testing environment without active socket connections, `ActiveConnections` in `Totals` is 0 while listening sockets are populated in `Connections`.
- **Scope:** This audit verifies backend telemetry and endpoints for Milestone M1. Frontend UI implementations (`Networks.tsx` and `NetworkFlow.tsx`) belong to Milestones M2 and M3.

---

## 4. Conclusion

**Verdict: CLEAN**

No integrity violations were detected. Milestone M1 work products demonstrate authentic logic, mathematical correctness, robust error handling, full backward compatibility, clean compilation, and 100% test pass rates.

---

## 5. Verification Method

To independently reproduce this forensic audit:

1. **Verify No Hardcoded Artifacts:**
   ```powershell
   Get-ChildItem -Path backend -Recurse -Include *.log,*result*,*output*,*fixture* -File
   ```
   *Expected:* No output (empty).

2. **Execute Full Backend Test Suite:**
   ```powershell
   cd c:\Users\phili\Documents\GitHub\Orbit\backend
   go test -v ./internal/api ./internal/netflow
   go test ./...
   ```
   *Expected:* All packages report `ok` and `PASS`.

3. **Execute Full Backend Compilation:**
   ```powershell
   cd c:\Users\phili\Documents\GitHub\Orbit\backend
   go build ./...
   ```
   *Expected:* Exit code 0 with no errors.
