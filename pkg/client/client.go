package client

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pb "github.com/tvandinther/gitops-manager/gen/go"
)

type Client struct {
	serverHost string
	grpcClient pb.GitOpsClient
	connection *grpc.ClientConn
}

type RequestOptions struct {
	ServerHost        string
	SecureTransport   bool
	TargetRepository  *string
	Environment       *string
	AppName           *string
	UpdateIdentifier  *string
	DryRun            *bool
	AutoReview        *bool
	ManifestDirectory string
	SourceRepository  *string
	CommitSHA         *string
	Actor             *string
	SourceAttributes  *string
}

func New(serverHostname string, secure bool) *Client {
	client := &Client{
		serverHost: serverHostname,
	}

	var creds credentials.TransportCredentials
	if secure {
		creds = credentials.NewClientTLSFromCert(nil, "")
	} else {
		creds = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(
		client.serverHost,
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             2 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		log.Fatalf("could not create client: %v", err)
	}
	client.connection = conn

	client.grpcClient = pb.NewGitOpsClient(conn)

	return client
}

func ParseRequestOptions() *RequestOptions {
	options := &RequestOptions{
		TargetRepository: flag.String("target-repository", "", "Name of the target configuration repository"),
		Environment:      flag.String("env", "", "Target environment"),
		AppName:          flag.String("app", "", "Application name"),
		UpdateIdentifier: flag.String("update-id", "", "Update identifier (e.g., git branch)"),
		DryRun:           flag.Bool("dry-run", true, "Enable dry-run mode"),
		AutoReview:       flag.Bool("auto-review", false, "Enable automatic completion of reviews"),
		SourceRepository: flag.String("source-repository", "", "Source repository"),
		CommitSHA:        flag.String("commit-sha", "", "Commit SHA"),
		Actor:            flag.String("actor", "", "Actor"),
		SourceAttributes: flag.String("source-attributes", "", "Source attributes"),
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		log.Fatal("usage: program [options] <manifests-directory> <gitops-server-url>")
	}
	options.ManifestDirectory = args[0]
	options.ServerHost = args[1]

	options.SecureTransport = getEnvBool("GITOPS_SECURE", false)

	return options
}

func (c *Client) Dispose() {
	c.connection.Close()
}

func getEnvBool(key string, defaultVal bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		log.Fatalf("invalid boolean value for %s: %v", key, err)
	}
	return val
}
