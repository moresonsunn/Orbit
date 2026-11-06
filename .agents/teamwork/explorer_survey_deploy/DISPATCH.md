## 2026-11-06T00:59:07Z
You are explorer_survey_deploy, an Explorer agent.
Your working directory is: c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_deploy

MANDATORY FIRST STEP:
Read c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\ORIGINAL_REQUEST.md before starting any work.

Reference: https://github.com/moresonsunn/Lynx

SCOPE:
1. Inspect Orbit's root directory: `docker-compose.yml`, `docker-compose.prod.yml`, `Dockerfile`, `README.md`, environment variables, config files.
2. Inspect Lynx's docker-compose, `docker run` quickstart, and README layout (badges, architecture, feature breakdown, env reference).
3. Determine exact requirements for:
   - `docker-compose.yml` (standard development/standalone deployment)
   - `docker-compose.prod.yml` (production server deployment with `/var/run/docker.sock:/var/run/docker.sock`, `/data` volume mount, restart policy `unless-stopped` or `always`, environment variables, port mapping `8080:8080`).
   - `README.md`:
     - Header badges: CI build status, Docker image/pulls, License, Release.
     - Architecture overview & feature breakdown (Containers, Networks, IPAM, Network Flow, GPU, Images, Volumes, Events, Logs, Auth).
     - One-line quickstart instructions for `docker run` and `docker compose`.
     - Comprehensive environment variable reference table.
4. Write a detailed report to `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_deploy\handoff.md`.

When done, write handoff.md and send a completion message with the path to the orchestrator.

