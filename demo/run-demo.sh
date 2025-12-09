#!/bin/bash
set -e

# Color codes for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PROMETHEUS_OPERATOR_VERSION="v0.83.0"
K8S_DEMO_NAMESPACE="cc-intel-reg-svc-demo"

check_prerequisites() {
  echo -e "${YELLOW}Checking prerequisites...${NC}"

  # Check kubectl (common for both deployment methods)
  if ! command -v kubectl &>/dev/null; then
    echo -e "${RED}Error: kubectl is not installed.${NC}"
    exit 1
  fi

  # Check curl (common for both deployment methods)
  if ! command -v curl &>/dev/null; then
    echo -e "${RED}Error: curl is not installed.${NC}"
    exit 1
  fi

  # Check deployment method specific tools
  if [ "$DEPLOYMENT_METHOD" == "helm" ]; then
    # Check helm
    if ! command -v helm &>/dev/null; then
      echo -e "${RED}Error: helm is not installed.${NC}"
      exit 1
    fi
  elif [ "$DEPLOYMENT_METHOD" == "compose" ]; then
    # Check docker and docker-compose
    if ! command -v docker &>/dev/null; then
      echo -e "${RED}Error: docker is not installed.${NC}"
      exit 1
    fi

  else
    echo -e "${RED}Error: Invalid deployment method. Use 'helm' or 'compose'.${NC}"
    exit 1
  fi

  echo -e "${GREEN}All prerequisites are met.${NC}"
}
# Check for required tools
check_prerequisites() {
  echo -e "${YELLOW}Checking prerequisites...${NC}"

  # Check kubectl
  if ! command -v kubectl &>/dev/null; then
    echo -e "${RED}Error: kubectl is not installed.${NC}"
    exit 1
  fi

  # Check helm
  if ! command -v helm &>/dev/null; then
    echo -e "${RED}Error: helm is not installed.${NC}"
    exit 1
  fi

  # Check docker
  if ! command -v docker &>/dev/null; then
    echo -e "${RED}Error: docker is not installed.${NC}"
    exit 1
  fi

  echo -e "${GREEN}All prerequisites are met.${NC}"
}

# Validate .env file
validate_env_file() {
  if [ ! -f .env ]; then
    echo -e "${RED}Error: .env file not found. Please create and configure it first.${NC}"
    exit 1
  fi

  # Source the .env file
  source .env

  # Check if required variables are set
  if [ -z "$REGISTRY_USERNAME" ] ||
    [ -z "$REGISTRY_ACCESS_TOKEN" ] ||
    [ -z "$REGISTRY" ] ||
    [ -z "$CC_IPR_REGISTRATION_INTERVAL_MINUTES" ] ||
    [ -z "$CC_SERVICE_PORT" ] ||
    [ -z "$VERSION" ] ||
    [ -z "$REGISTRY_EMAIL" ]; then
    echo -e "${RED}Error: Please fill in all required variables in the .env file.${NC}"
    exit 1
  fi
}

show_menu() {
  clear
  echo -e "${GREEN}===== CC Intel Platform Registration Deployment =====\n${NC}"
  echo -e "${YELLOW}Select Deployment Method:${NC}"
  echo "1. Deploy with Helm (Recommended)"
  echo "2. Deploy with Docker Compose"
  echo "3. Exit"
  echo -en "\n${YELLOW}Enter your choice [1-3]: ${NC}"
}

deploy_demo() {
  # Validate .env file
  validate_env_file || exit 1

  # Set deployment method based on user choice
  case $DEPLOYMENT_METHOD in
  helm)
    check_prerequisites || exit 1
    deploy_helm
    ;;
  compose)
    check_prerequisites || exit 1
    deploy_docker_compose
    ;;
  *)
    echo -e "${RED}Invalid deployment method.${NC}"
    exit 1
    ;;
  esac
}

# Main deployment function
deploy_helm() {
  check_prerequisites

  # Create namespace
  echo -e "${YELLOW}Creating namespace $K8S_DEMO_NAMESPACE...${NC}"
  kubectl create namespace $K8S_DEMO_NAMESPACE

  # Create docker registry secret
  echo -e "${YELLOW}Creating registry pull secret...${NC}"
  kubectl create secret docker-registry reg-svc-pull-secret \
    --docker-server=$REGISTRY \
    --docker-username="$REGISTRY_USERNAME" \
    --docker-password="$REGISTRY_ACCESS_TOKEN" \
    --docker-email="$REGISTRY_EMAIL" \
    --namespace $K8S_DEMO_NAMESPACE

  echo -e "${YELLOW}Installing Prometheus Operator...${NC}"
  curl -sL https://github.com/prometheus-operator/prometheus-operator/releases/download/${PROMETHEUS_OPERATOR_VERSION}/bundle.yaml | kubectl create -f -


  kubectl wait --for=condition=Ready pods -l  app.kubernetes.io/name=prometheus-operator

  # Install Helm chart
  echo -e "${YELLOW}Installing Helm chart...${NC}"
  helm install reg-svc --namespace $K8S_DEMO_NAMESPACE charts/cc-intel-platform-registration \
    --set "fullnameOverride=reg-svc" \
    --set image.repository="${REGISTRY}/cc-intel-platform-registration" \
    --set image.tag="${VERSION}" \
    --set createPrometheusPodMonitor=true \
    --set registrationIntervalInMinutes="$CC_IPR_REGISTRATION_INTERVAL_MINUTES" \
    --set imagePullSecrets[0].name=reg-svc-pull-secret \
    --set sconeSGXPlugin.enabled=true \
    --set sconeSGXPlugin.image.repository="${REGISTRY}/sgx-plugin" \
    --wait

  # Check registration service pod
  echo -e "${YELLOW}Checking registration service pod...${NC}"
  kubectl get pod -n $K8S_DEMO_NAMESPACE

  # Deploy Grafana and Prometheus
  echo -e "${YELLOW}Deploying Grafana and Prometheus...${NC}"
  pushd demo/demo-manifests
  kubectl apply -f namespace.yaml
  kubectl apply -f prometheus.yaml
  kubectl apply -f grafana.yaml
  popd

  # Final status check
  echo -e "${GREEN}Deployment completed successfully!${NC}"
  echo -e "${YELLOW}Grafana Credentials:${NC}"
  echo "Username: admin"
  echo "Password: admin"

  # Provide port-forward instructions
  echo -e "\n${YELLOW}To access Grafana:${NC}"
  echo "Run: kubectl port-forward -n monitoring services/grafana 3000"
  echo "Then navigate to http://localhost:3000"
}

function cleanup {
  rm -rf "$WORK_DIR"
  echo "Deleted temp working directory $WORK_DIR"
}

trap cleanup EXIT

deploy_docker_compose() {

  # Create necessary directories
  WORK_DIR=$(mktemp -d)

  # register the cleanup function to be called on the EXIT signal

  # Create Prometheus configuration
  cat >${WORK_DIR}/prometheus.yml <<EOF
global:
  scrape_interval: 5s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "cc-intel-platform-registration"
    scrape_interval: 5s
    metrics_path: "/metrics"
    static_configs:
      - targets: ["cc-intel-platform-registration:8080"]
EOF

  # Create Grafana datasource configuration
  cat >${WORK_DIR}/grafana_datasources.yml <<EOF
apiVersion: 1

datasources:
  - name: prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
EOF
  cat >${WORK_DIR}/main_dashboard.yml <<EOF
apiVersion: 1
providers:
  - name: "Default"
    orgId: 1
    folder: ""
    type: file
    disableDeletion: true
    editable: false
    options:
      path: /etc/grafana/provisioning/dashboards
EOF

  cat >${WORK_DIR}/reg_svc.json <<EOF
{
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": {
          "type": "grafana",
          "uid": "-- Grafana --"
        },
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "fiscalYearStartMonth": 0,
  "graphTooltip": 0,
  "id": 1,
  "links": [],
  "panels": [
    {
      "datasource": {
        "uid": "prometheus"
      },
      "description": "Registration status code",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "thresholds"
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "red",
                "value": 80
              }
            ]
          }
        },
        "overrides": []
      },
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 0,
        "y": 0
      },
      "id": 1,
      "options": {
        "colorMode": "value",
        "graphMode": "area",
        "justifyMode": "auto",
        "orientation": "auto",
        "percentChangeColorMode": "standard",
        "reduceOptions": {
          "calcs": [
            "lastNotNull"
          ],
          "fields": "",
          "values": false
        },
        "showPercentChange": false,
        "textMode": "auto",
        "wideLayout": true
      },
      "pluginVersion": "11.5.1",
      "targets": [
        {
          "disableTextWrap": false,
          "editorMode": "builder",
          "expr": "service_status_code",
          "fullMetaSearch": false,
          "includeNullMetadata": true,
          "legendFormat": "__auto",
          "range": true,
          "refId": "A",
          "useBackend": false
        }
      ],
      "title": "Registration Status",
      "type": "stat"
    }
  ],
  "preload": false,
  "refresh": "",
  "schemaVersion": 40,
  "tags": [],
  "templating": {
    "list": []
  },
  "time": {
    "from": "now-5m",
    "to": "now"
  },
  "timepicker": {},
  "timezone": "browser",
  "title": "Registration Service",
  "uid": "fec2fclyjtiwwf",
  "version": 2,
  "weekStart": ""
}
EOF

  # Start the containers
  echo "Starting containers..."

  cp .env $WORK_DIR/.env

  echo "WORK_DIR=$WORK_DIR" >>$WORK_DIR/.env

  pushd demo

  if command docker compose version &>/dev/null; then
    docker compose --env-file ${WORK_DIR}/.env up -d
  else
    echo "Docker compose requires sudo."
    sudo docker compose --env-file ${WORK_DIR}/.env up -d
  fi
  popd

  echo "Deployment complete!"
  echo "Access Prometheus at http://localhost:9090"
  echo "Access Grafana at http://localhost:3000 (admin/admin)"

}

# Interactive menu loop
while true; do
  show_menu
  read -r choice

  case $choice in
  1)
    DEPLOYMENT_METHOD=helm
    deploy_demo
    exit 0
    ;;
  2)
    DEPLOYMENT_METHOD=compose
    deploy_demo
    exit 0
    ;;
  3)
    echo -e "${GREEN}Exiting deployment script.${NC}"
    exit 0
    ;;
  *)
    echo -e "${RED}Invalid option. Please try again.${NC}"
    echo -en "\n${GREEN}Press Enter to continue...${NC}"
    read -r
    ;;
  esac
done
