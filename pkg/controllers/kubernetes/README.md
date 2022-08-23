# Kubernetes Controller

This folder contains the Kubernetes controller that allows Releases to be submitted as Kubernetes
custom resources.

[Kubebuilder](https://book.kubebuilder.io/) is used to scaffold the controller.

## Building the Resources
Kubebuilder can automatically generate CRDs and other asserts for the controller, to generate these assets
run the following command:

```
make kustomize_build
```

The compiled assets are output to the `kustomize_build` directory.
