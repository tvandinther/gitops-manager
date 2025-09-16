export LOG_LEVEL := "debug"
export GITEA_HOST := "http://localhost:3000"
export GITEA_USER := "gitops-manager"
export GITEA_ACCESS_TOKEN := "58359fe2e5db3884c70d8646971ebf5cae21fd63"

set dotenv-load := true

default:
  @just --choose

gen:
  buf generate proto

up *ARGS:
    docker compose up {{ ARGS }}

down:
    docker compose down

run-server:
  go run cmd/server/main.go

run-client:
  go run cmd/client/main.go \
    -target-repository $GITEA_HOST/admin/config.git \
    -env test \
    -dry-run=false \
    -auto-review=false \
    -app foo \
    -update-id test \
    -source-attributes '{"test": "foo"}' \
    example-files \
    localhost:50051 
