# Processors

- [Mutators](#mutators)
    - [Helm Hook To Argo CD Sync Hook](#helm-hook-to-argo-cd-sync-hook)
    - [Custom Mutators](#custom-mutators)
- [Validators](#validators)
    - [Empty Files](#empty-files)
    - [Custom Validators](#custom-validators)

## Mutators
*You can implement your own by creating a type that satisfies the `gitops.Mutator` interface.*

### Helm Hook To Argo CD Sync Hook
This mutator converts Helm hooks to Argo CD sync hooks. It looks for the annotation `helm.sh/hook` and converts it to the corresponding Argo CD sync hook annotation as per the Argo CD [docs](https://argo-cd.readthedocs.io/en/stable/user-guide/helm/#helm-hooks).

```go
flow.AddMutator(&mutators.HelmHooksToArgoCD{})
```

### Custom Mutators
You can create custom mutators by implementing the `gitops.Mutator` interface. This allows you to define specific mutation logic that suits your requirements. Write your mutations to the provided `io.Writer` in the `MutateFile` method. Any data written to to the writter will overwrite the input data and passed to the next mutator in the chain. If nothing is written to the writer, the next mutator in the chain will receive the original input.

## Validators
*You can implement your own by creating a type that satisfies the `gitops.Validator` interface.*

### Empty Files
This validator checks for empty files in the manifests. Empty files are often a result of misconfigurations or errors in the rendering process and can lead to issues when applying configuration. They could also indicate a catastrophic failure in the rendering pipeline. Using this validator helps to protect against such scenarios by ensuring that all files contain valid content before they are committed to the repository.

```go
flow.AddValidator(&validators.EmptyFiles{})
```

### Custom Validators
You can create custom validators by implementing the `gitops.Validator` interface. This allows you to define specific validation logic that suits your requirements. Read the file from the provided `io.Reader` in the `ValidateFile` method. Return a `gitops.ValidationResult` indicating whether the file is valid or not, along with a slice of applicable errors in the case where a validation is not valid.
