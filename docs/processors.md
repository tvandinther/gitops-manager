# Processors

- [Mutators](#mutators)
    - [Helm Hook To Argo CD Sync Hook](#helm-hook-to-argo-cd-sync-hook)
- [Validators](#validators)

## Mutators

*You can implement your own by creating a type that satisfies the `gitops.Mutator` interface.*

### Helm Hook To Argo CD Sync Hook
This mutator converts Helm hooks to Argo CD sync hooks. It looks for the annotation `helm.sh/hook` and converts it to the corresponding Argo CD sync hook annotation as per the Argo CD [docs](https://argo-cd.readthedocs.io/en/stable/user-guide/helm/#helm-hooks).

```go
flow.AddMutator(&mutators.HelmHooksToArgoCD{})
```

## Validators

*There are currently no built-in validators. You can implement your own by creating a type that satisfies the `gitops.Validator` interface.*
