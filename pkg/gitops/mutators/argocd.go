package mutators

import (
	"context"
	"fmt"
	"io"

	yamlUtil "github.com/tvandinther/gitops-manager/pkg/util"
	"gopkg.in/yaml.v3"
)

// This mutator converts Helm hooks to equivalent Argo CD sync hooks
type HelmHooksToArgoCD struct{}

func (h *HelmHooksToArgoCD) GetTitle() string {
	return "Helm Hooks to Argo CD sync hooks"
}

func (h *HelmHooksToArgoCD) MutateFile(ctx context.Context, inputFile io.Reader, outputFile io.Writer, sendMsg func(string)) error {
	var root yaml.Node
	err := yaml.NewDecoder(inputFile).Decode(&root)
	// If empty file, do nothing
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to parse file as YAML: %w", err)
	}

	for _, doc := range root.Content {
		metadata := yamlUtil.GetOrCreateMap(doc, "metadata")
		annotations := yamlUtil.GetOrCreateMap(metadata, "annotations")

		convertHelmHooks(annotations)
	}

	err = yaml.NewEncoder(outputFile).Encode(&root)
	if err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return nil
}

var helmToArgoCdHook map[string]string = map[string]string{
	"crd-install":  "PreSync",
	"pre-install":  "PreSync",
	"pre-upgrade":  "PreSync",
	"post-upgrade": "PostSync",
	"post-install": "PostSync",
	"post-delete":  "PostDelete",
}

func convertHelmHooks(annotations *yaml.Node) {
	var indicesToDelete []int

	for i := 0; i < len(annotations.Content); i += 2 {
		if annotations.Content[i].Value == "helm.sh/hook" {
			helmHook := annotations.Content[i+1].Value
			argocdHook, ok := helmToArgoCdHook[helmHook]
			if !ok {
				continue
			}
			yamlUtil.SetMappingValue(annotations, "argocd.argoproj.io/hook", argocdHook)
			indicesToDelete = append(indicesToDelete, i)

			break
		}
	}

	yamlUtil.DeleteMappingKeysByIndices(annotations, indicesToDelete)
}
