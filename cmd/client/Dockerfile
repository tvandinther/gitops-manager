ARG APP_NAME="client"

FROM golang:1.24.5-alpine AS builder

ARG APP_NAME

ENV USER=gitops
ENV UID=10001

RUN adduser \    
--disabled-password \    
--gecos "" \    
--home "/${USER}" \    
--shell "/sbin/nologin" \    
--no-create-home \    
--uid "${UID}" \    
"${USER}"

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /${APP_NAME} cmd/${APP_NAME}/*.go

FROM scratch

ARG APP_NAME

ENV APP_NAME=${APP_NAME}

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /${APP_NAME} /home/gitops/main

USER gitops:gitops

ENTRYPOINT ["/home/gitops/main"]
