package mutators

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"

	yamlUtil "github.com/tvandinther/gitops-manager/pkg/util"
	"gopkg.in/yaml.v3"
)

// This mutator converts Helm hooks to equivalent Argo CD sync hooks
type HelmHooksToArgoCD struct{}

func (h *HelmHooksToArgoCD) GetTitle() string {
	return "Helm Hooks to Argo CD sync hooks"
}

func (h *HelmHooksToArgoCD) Mutate(ctx context.Context, dir string, setError func(e error), next func(), sendMsg func(string)) {
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !d.IsDir() {
			log := slog.With("path", path)
			log.Debug("mutating file")

			root, err := yamlUtil.ParseFileToYamlNode(path)
			if err != nil {
				return fmt.Errorf("failed to parse file as YAML: %w", err)
			}

			for _, doc := range root.Content {
				metadata := yamlUtil.GetOrCreateMap(doc, "metadata")
				annotations := yamlUtil.GetOrCreateMap(metadata, "annotations")

				convertHelmHooks(annotations)
			}

			err = yamlUtil.WriteToFile(path, root)
			if err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			log.Debug("file mutated")

			return nil
		}

		return nil
	})

	setError(err)
	next()
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
