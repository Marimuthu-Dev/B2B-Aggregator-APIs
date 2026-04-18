#!/usr/bin/env bash
set -e
# Run from the Go module directory: .../B2B-Aggregator-APIs/src
#
# WebJobs expect the entry script at zip root to be named run.sh.
# Source in repo: run-fitness-worker.sh → copy as run.sh into the zip.
echo "1. Creating fitness-job.zip (excluding chrome-linux64)..."
rm -f run.sh fitness-job.zip
cp run-fitness-worker.sh run.sh
chmod +x run.sh
zip -r fitness-job.zip fitness-worker run.sh settings.job templates/
rm -f run.sh

echo "2. Deploying WebJob directly to App Service via Az CLI..."
# We push the zip specifically to the Kudu WebJob continuous folder.
az webapp deploy \
  --resource-group um-staging-rg-appservice \
  --name um-staging-worker-process \
  --src-path fitness-job.zip \
  --target-path /home/site/wwwroot/app_data/jobs/triggered/fitness-job \
  --type zip

echo "Deployment complete! You should see it in the Azure portal now."
