#!/bin/bash
# filepath: /home/arbion/Dev/posta/run-dev.sh

# Default values
COMPOSE_FILE="./deploy/docker-compose/dev.yml"
COMMAND="up"

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -f|--file)
      COMPOSE_FILE="$2"
      shift
      shift
      ;;
    -c|--command)
      COMMAND="$2"
      shift
      shift
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [-f|--file <compose-file>] [-c|--command <command>]"
      exit 1
      ;;
  esac
done

# Run the Docker Compose command
docker compose -f "$COMPOSE_FILE" $COMMAND