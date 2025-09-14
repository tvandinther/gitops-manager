package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/tvandinther/gitops-manager/pkg/client"
	"github.com/tvandinther/gitops-manager/pkg/client/request"
	"github.com/tvandinther/gitops-manager/pkg/gitops"
)

func main() {
	requestOptions := client.ParseRequestOptions()
	gitopsClient := client.New(requestOptions.ServerHost, requestOptions.SecureTransport)
	defer gitopsClient.Dispose()

	timeout := 30 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var sourceAttributesMap map[string]any
	if *requestOptions.SourceAttributes == "" {
		sourceAttributesMap = make(map[string]any)
	} else {
		err := json.Unmarshal([]byte(*requestOptions.SourceAttributes), &sourceAttributesMap)
		if err != nil {
			log.Fatalf("failed to unmarshal JSON from source attributes: %v", err)
		}
	}

	absDirPath, err := filepath.Abs(requestOptions.ManifestDirectory)
	if err != nil {
		log.Fatalf("error parsing manifests directory: %s", requestOptions.ManifestDirectory)
	}

	totalFiles, err := client.CountFiles(absDirPath)
	if err != nil {
		log.Fatalf("error while counting files in manifest directory")
	}

	req := request.New(
		request.WithRepository(gitops.Repository{
			URL: *requestOptions.TargetRepository,
		}),
		request.WithEnvironment(*requestOptions.Environment),
		request.WithAppName(*requestOptions.AppName),
		request.WithUpdateIdentifier(*requestOptions.UpdateIdentifier),
		request.WithDryRun(*requestOptions.DryRun),
		request.WithAutoReview(*requestOptions.AutoReview),
		request.WithSource(request.RequestSource{
			Repository: gitops.Repository{
				URL: *requestOptions.SourceRepository,
			},
			Metadata: request.RequestSourceMetadata{
				CommitSHA:  *requestOptions.CommitSHA,
				Actor:      *requestOptions.Actor,
				Attributes: sourceAttributesMap,
			},
		}),
		request.WithTotalFiles(totalFiles),
	)
	client.PrettyPrintManifestRequest(req)

	rCtx, err := gitopsClient.SendRequest(req, client.WithContext(ctx))
	if err != nil {
		log.Fatalf("error sending request: %v", err)
	}

	err = rCtx.UploadDirectory(absDirPath)
	if err != nil {
		log.Fatalf("error uploading manifests: %v", err)
	}

	rCtx.Wait()

	for _, err := range rCtx.Errors {
		fmt.Printf("[ERROR] %v\n", err)
	}

	if len(rCtx.Errors) > 0 {
		os.Exit(1)
	}
}
