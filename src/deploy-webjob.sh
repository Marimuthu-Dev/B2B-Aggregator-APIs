#!/usr/bin/env bash
set -e

echo "1. Creating fitness-job.zip (excluding chrome-linux64)..."
rm -f fitness-job.zip
zip -r fitness-job.zip fitness-worker run.sh settings.job templates/

echo "2. Deploying WebJob directly to App Service via Az CLI..."
# We push the zip specifically to the Kudu WebJob continuous folder.
az webapp deploy \
  --resource-group um-staging-rg-appservice \
  --name um-staging-worker-process \
  --src-path fitness-job.zip \
  --target-path /home/site/wwwroot/app_data/jobs/triggered/fitness-job \
  --type zip

echo "Deployment complete! You should see it in the Azure portal now."
