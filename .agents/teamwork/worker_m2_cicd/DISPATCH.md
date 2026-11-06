## 2026-11-06T01:20:55Z
You are worker_m2_cicd, an implementation worker agent.
Your working directory is: c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_m2_cicd

MANDATORY FIRST STEP:
Read c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\ORIGINAL_REQUEST.md before starting any work.

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

REFERENCE FILES:
- `PROJECT.md` at repository root
- Explorer survey and specifications: `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_cicd\handoff.md`

WRITE OWNERSHIP:
You exclusively own:
- `.github/workflows/ci.yml`
- `.github/workflows/docker-publish.yml`
- `.github/workflows/release.yml`
You MUST NOT edit any frontend source files, backend source files, compose files, or `README.md`.

TASKS FOR MILESTONE M2:
Implement the complete GitHub Actions workflows modeled directly on Lynx (https://github.com/moresonsunn/Lynx) using the exact specifications in `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_cicd\handoff.md`:
1. Create `.github/workflows/ci.yml`:
   - Parallel jobs: `backend` (Go tests `go test -v ./...`, `go vet`, `go build`) and `frontend` (Node 22, `npm ci`, `npm run build`, asset check).
   - Triggers: push & pull_request on main/master, plus workflow_dispatch.
2. Create `.github/workflows/docker-publish.yml`:
   - Triggers: push on main/master and tags `v*`, plus workflow_dispatch.
   - Multi-arch build for `linux/amd64` and `linux/arm64`.
   - Setup QEMU (`docker/setup-qemu-action@v3`) and Buildx (`docker/setup-buildx-action@v3`).
   - Dual registry login: Docker Hub (`moresonsunn/orbit`) using `DOCKERHUB_USERNAME` & `DOCKERHUB_TOKEN` secrets (with `DOCKER_USERNAME`/`DOCKER_TOKEN` fallback) and GHCR (`ghcr.io`) using `GITHUB_TOKEN`.
   - Docker metadata action for multi-registry tags (latest, branch, tag, semver, sha).
   - Build-push action with `type=gha` caching and multi-arch platforms.
3. Create `.github/workflows/release.yml`:
   - Triggers on push on tags `v*`.
   - Generates GitHub release via `softprops/action-gh-release@v2` with auto release notes.

VERIFICATION:
Verify YAML formatting and syntax.
Run E2E workflow validation test:
`cd tests/e2e && go test -v -run TestTier1_DeployUI_FeatureCoverage/D03 ./...`
Document commands and results in `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_m2_cicd\handoff.md` and send a completion message to the orchestrator.
