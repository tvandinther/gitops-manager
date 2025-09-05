default:
  @just --choose

gen:
  buf generate proto

up *ARGS:
    docker compose up {{ ARGS }}

down:
    docker compose down

run-server:
  LOG_LEVEL=DEBUG go run cmd/server/main.go

run-client:
  LOG_LEVEL=DEBUG go run cmd/client/main.go -target-repository http://localhost:3000/admin/config.git -env test -dry-run -auto-review -app foo -source-attributes '{"test": "foo"}' example-files localhost:50051 
