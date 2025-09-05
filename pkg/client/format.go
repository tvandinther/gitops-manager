package client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	pb "github.com/tvandinther/gitops-manager/gen/go"
)

type JobSummary struct {
	Message           string              `json:"message"`
	UpdatedFilesCount int32               `json:"updated_files_count"`
	DryRun            bool                `json:"dry_run"`
	Review            *ReviewSummary      `json:"review"`
	Environment       *EnvironmentSummary `json:"environment"`
}

type ReviewSummary struct {
	Created   bool   `json:"created"`
	Url       string `json:"url"`
	Completed bool   `json:"completed"`
}

type EnvironmentSummary struct {
	Repository string `json:"repository"`
	Name       string `json:"name"`
	RefName    string `json:"ref_name"`
}

func PrintProgress(p *pb.Progress) {
	switch p.Kind {
	case pb.ProgressKind_HEADING:
		fmt.Printf("\x1b[1m* %s \033[0m\n", strings.ToUpper(p.Status))
	case pb.ProgressKind_PROGRESS:
		fmt.Printf("→ %s\n", p.Status)
	case pb.ProgressKind_SUCCESS:
		fmt.Printf("\033[32m✔ %s\033[0m\n", p.Status)
	case pb.ProgressKind_FAILURE:
		fmt.Printf("\033[31m✖ %s\033[0m\n", p.Status)
	default:
		fmt.Println(p.Status)
	}
}

func (s *JobSummary) FromProto(p *pb.Summary) {
	s.Message = p.GetMessage()
	s.UpdatedFilesCount = p.GetUpdatedFilesCount()
	s.DryRun = p.GetDryRun()
	s.Review = &ReviewSummary{
		Created:   p.GetReview().GetCreated(),
		Completed: p.GetReview().GetCompleted(),
		Url:       p.GetReview().GetUrl(),
	}
	s.Environment = &EnvironmentSummary{
		Name:       p.GetEnvironment().GetName(),
		RefName:    p.GetEnvironment().GetRefName(),
		Repository: p.GetEnvironment().GetRepository().GetUrl(),
	}
}

func printKV(indent int, label, value string) {
	padding := strings.Repeat("  ", indent)
	fmt.Printf("%s%-18s : %s\n", padding, label, value)
}

func PrettyPrintManifestRequest(req *pb.ManifestRequest) {
	if req == nil || req.Content == nil {
		fmt.Println("ManifestRequest: <nil>")
		return
	}

	meta := req.GetMetadata()
	if meta == nil {
		fmt.Println("Manifest Metadata: <nil>")
		return
	}

	fmt.Println("Manifest Update Request:")

	printKV(1, "Environment", meta.Environment)
	printKV(1, "App Name", meta.AppName)
	printKV(1, "Update Branch", meta.UpdateIdentifier)
	printKV(1, "Dry Run", fmt.Sprintf("%v", meta.DryRun))
	printKV(1, "Auto Review", fmt.Sprintf("%v", meta.AutoReview))
	printKV(1, "Config Repository", meta.ConfigRepository.Url)

	sourceMetadataAttributes := make(map[string]any)
	for k, v := range meta.Source.Metadata.GetAttributes() {
		sourceMetadataAttributes[k] = v.AsInterface()
	}

	jsonAttributes, err := json.Marshal(sourceMetadataAttributes)
	if err != nil {
		// TODO
	}

	if meta.Source != nil {
		fmt.Println("  Source:")
		printKV(2, "Repository", meta.Source.Repository.Url)
		printKV(2, "Commit SHA", meta.Source.Metadata.CommitSha)
		printKV(2, "Actor", meta.Source.Metadata.Actor)
		printKV(2, "Attributes", string(jsonAttributes))
	}
	fmt.Println()
}

func PrettyPrintJSONBlock(title string, jsonBytes []byte) error {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return err
	}

	fmt.Printf("%s:\n", title)
	printIndentedBlockSorted(data, 1)
	return nil
}

func printIndentedBlockSorted(data map[string]interface{}, indentLevel int) {
	var flatKeys, nestedKeys []string
	labelWidth := 0

	for k, v := range data {
		switch v.(type) {
		case map[string]interface{}:
			nestedKeys = append(nestedKeys, k)
		default:
			flatKeys = append(flatKeys, k)
			if len(k) > labelWidth {
				labelWidth = len(k)
			}
		}
	}
	sort.Strings(flatKeys)
	sort.Strings(nestedKeys)

	indent := strings.Repeat("  ", indentLevel)

	for _, key := range flatKeys {
		val := data[key]
		fmt.Printf("%s%-*s : %v\n", indent, labelWidth, key, val)
	}

	for _, key := range nestedKeys {
		val := data[key].(map[string]interface{})
		fmt.Printf("%s%s:\n", indent, key)
		printIndentedBlockSorted(val, indentLevel+1)
	}
}
