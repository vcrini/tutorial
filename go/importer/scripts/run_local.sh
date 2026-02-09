#!/bin/sh
set -e

cd "$(dirname "$0")/.."

export AWS_REGION=${AWS_REGION:-eu-west-1}
export SOC_COD=${SOC_COD:-045}
export DATABASE_NAME=${DATABASE_NAME:-gthsdu_dev_factory2erp}
export DATABASE_FOLDER=${DATABASE_FOLDER:-/data/database/}
export S3_BUCKET_NAME=${S3_BUCKET_NAME:-proclient-gthsdu-dev-factory1sbxerp}
export S3_SOURCE_FOLDER=${S3_SOURCE_FOLDER:-data/landing_zone/}
export S3_ARCHIVE_FOLDER=${S3_ARCHIVE_FOLDER:-data/archive/}
export S3_ARCHIVE_INVALID_FOLDER=${S3_ARCHIVE_INVALID_FOLDER:-data/archive/invalid_documents/}
export BATCH_SIZE=${BATCH_SIZE:-10}
export BATCH_WEIGHT_IN_BYTE=${BATCH_WEIGHT_IN_BYTE:-1048576}
export INTERVAL_IN_MINUTE=${INTERVAL_IN_MINUTE:-1}
export SQS_QUEUE=${SQS_QUEUE:-gthsdu-dev-factory2sbxerpwhmovementdpl0.fifo}
export REINIT=${REINIT:-false}
export DELAY_IN_MINUTE=${DELAY_IN_MINUTE:-1}

GOCACHE=${GOCACHE:-/tmp/go-build} go build ./cmd/whmovement

./whmovement
