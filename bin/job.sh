#!/usr/bin/env bash

## - Idea borrowed from
# - https://github.com/Hearst-Hatchery/appliance-api/blob/master/bin/run-worker.sh
# - https://github.com/Hearst-Hatchery/appliance-api/blob/master/k8s/appliance-worker/deployment.yml#L38

./go-crawler --env ${ENV_MODE} --job-type ${JOBTYPE} --worker_id ${WORKER_ID} --job --rest