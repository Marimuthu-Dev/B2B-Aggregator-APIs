#!/usr/bin/env bash
# Deploy B2B Aggregator API container to Azure App Service (um-staging-api)
# Run from repo root: ./scripts/deploy-container.sh
#
# Prerequisites:
#   - Azure CLI (az) logged in: az login
#   - Docker installed and running
#   - ACR created (or set CREATE_ACR=1 to create one)
#
# Set these before running (or export in shell):
#   ACR_NAME        - Azure Container Registry name (e.g. umstagingacr)
#   IMAGE_NAME      - Image name (default: b2b-aggregator-api)
#   IMAGE_TAG       - Tag (default: latest)
#   APP_NAME        - App Service name (default: um-staging-api)
#   RESOURCE_GROUP  - Resource group (default: um-staging-rg-appservice)

set -e

ACR_NAME="${ACR_NAME:-}"
IMAGE_NAME="${IMAGE_NAME:-b2b-aggregator-api}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
APP_NAME="${APP_NAME:-um-staging-api}"
RESOURCE_GROUP="${RESOURCE_GROUP:-um-staging-rg-appservice}"
CREATE_ACR="${CREATE_ACR:-0}"

if [ -z "$ACR_NAME" ]; then
  echo "Error: ACR_NAME is required. Example: export ACR_NAME=umstagingacr"
  echo "  ACR names must be globally unique, 5-50 chars, alphanumeric only."
  exit 1
fi

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

echo "=== Building Docker image ==="
docker build -t "${IMAGE_NAME}:${IMAGE_TAG}" .

echo "=== Logging in to Azure Container Registry ==="
az acr login --name "$ACR_NAME"

echo "=== Tagging image for ACR ==="
# ACR may be in a different resource group; set ACR_RESOURCE_GROUP if needed
ACR_RG="${ACR_RESOURCE_GROUP:-$RESOURCE_GROUP}"
ACR_LOGIN_SERVER=$(az acr show --name "$ACR_NAME" --resource-group "$ACR_RG" --query loginServer -o tsv 2>/dev/null || true)
if [ -z "$ACR_LOGIN_SERVER" ]; then
  ACR_LOGIN_SERVER="${ACR_NAME}.azurecr.io"
fi
docker tag "${IMAGE_NAME}:${IMAGE_TAG}" "${ACR_LOGIN_SERVER}/${IMAGE_NAME}:${IMAGE_TAG}"

echo "=== Pushing image to ACR ==="
docker push "${ACR_LOGIN_SERVER}/${IMAGE_NAME}:${IMAGE_TAG}"

echo "=== Ensuring ACR admin user is enabled (required for App Service pull) ==="
az acr update --name "$ACR_NAME" --resource-group "$ACR_RG" --admin-enabled true --output none 2>/dev/null || true

echo "=== Retrieving ACR credentials ==="
ACR_USER=$(az acr credential show --name "$ACR_NAME" --resource-group "$ACR_RG" --query username -o tsv)
ACR_PASS=$(az acr credential show --name "$ACR_NAME" --resource-group "$ACR_RG" --query "passwords[0].value" -o tsv)
if [ -z "$ACR_USER" ] || [ -z "$ACR_PASS" ]; then
  echo "ERROR: Could not get ACR credentials. Ensure admin user is enabled:"
  echo "  az acr update -n $ACR_NAME --resource-group $ACR_RG --admin-enabled true"
  exit 1
fi

echo "=== Configuring App Service to use the image ==="
az webapp config container set \
  --name "$APP_NAME" \
  --resource-group "$RESOURCE_GROUP" \
  --container-image-name "${ACR_LOGIN_SERVER}/${IMAGE_NAME}:${IMAGE_TAG}" \
  --container-registry-url "https://${ACR_LOGIN_SERVER}" \
  --container-registry-user "$ACR_USER" \
  --container-registry-password "$ACR_PASS"

echo "=== Restarting App Service ==="
az webapp restart --name "$APP_NAME" --resource-group "$RESOURCE_GROUP"

echo ""
echo "Done. API should be available at: https://${APP_NAME}.azurewebsites.net"
echo "Check health: https://${APP_NAME}.azurewebsites.net/ping"
echo ""
echo "If the app fails, set Application settings (DB_PASSWORD, JWT_SECRET, etc.) in:"
echo "  Azure Portal -> App Service -> Configuration -> Application settings"
