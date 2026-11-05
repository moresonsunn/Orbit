#!/bin/bash
# Scale test: Spawns 50 lightweight alpine containers to verify Orbit scalability under load
ACTION=${1:-up}

if [ "$ACTION" = "up" ]; then
  echo "Spawning 50 test containers for Orbit scale verification..."
  for i in $(seq 1 50); do
    docker run -d --name "orbit-scale-test-$i" \
      --label "orbit.test=scale" \
      --label "com.docker.compose.project=scale-bench" \
      alpine:latest sh -c "while true; do sleep 30; done" > /dev/null
  done
  echo "50 test containers started."
elif [ "$ACTION" = "down" ]; then
  echo "Cleaning up 50 test containers..."
  docker ps -q --filter "label=orbit.test=scale" | xargs -r docker rm -f > /dev/null
  echo "Cleanup complete."
fi

