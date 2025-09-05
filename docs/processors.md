# Processors

- [Mutators](#mutators)
    - [Helm Hook To Argo CD Sync Hook](#helm-hook-to-argo-cd-sync-hook)
- [Validators](#validators)
    - [Empty Files](#empty-files)

## Mutators
*You can implement your own by creating a type that satisfies the `gitops.Mutator` interface.*

### Helm Hook To Argo CD Sync Hook
This mutator converts Helm hooks to Argo CD sync hooks. It looks for the annotation `helm.sh/hook` and converts it to the corresponding Argo CD sync hook annotation as per the Argo CD [docs](https://argo-cd.readthedocs.io/en/stable/user-guide/helm/#helm-hooks).

```go
flow.AddMutator(&mutators.HelmHooksToArgoCD{})
```

## Validators
*You can implement your own by creating a type that satisfies the `gitops.Validator` interface.*

### Empty Files
This validator checks for empty files in the manifests. Empty files are often a result of misconfigurations or errors in the rendering process and can lead to issues when applying configuration. They could also indicate a catastrophic failure in the rendering pipeline. Using this validator helps to protect against such scenarios by ensuring that all files contain valid content before they are committed to the repository.

```go
flow.AddValidator(&validators.EmptyFiles{})
```
