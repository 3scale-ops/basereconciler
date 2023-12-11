# basereconciler

Basereconciler is an attempt to create a reconciler that can be imported an used in any controller-runtime based controller to perform the most common tasks a controller usually performs. It's a bunch of code that it's typically written again and again for every and each controller and that can be abstracted to work in a more generic way to avoid the repetition and improve code mantainability.
At the moment basereconciler can perform the following tasks:

* Get the custom resource and perform some common tasks on it:
  * Management of resource finalizer: some custom resources required more complex finalization logic. For this to happen a finalizer must be in place. Basereconciler can keep this finalizer in place and remove it when necessary during resource finalization.
  * Management of finalization logic: it checks if the resource is being finalized and executed the finalization logic passed to it if that is the case. When all finalization logic is completed it removes the finalizer on the custom resource.
* Reconcile resources owned by the custom resource: basreconciler can keep the owned resources of a custom resource in it's desired state. It works for any resource type, and only requires that the user configures how each specific resource type has to be configured. The resource reconciler only works in "update mode" right now, so any operation to transition a given resource from its live state to its desired state will be an Update. We might add a "patch mode" in the future.
* Reconcile custom resource status: if the custom resource implements a certain interface, basereconciler can also be in charge of reconciling the status.
* Resource pruner: when the reconciler stops seeing a certain resource, owned by the custom resource, it will prune them as it understands that the resource is no logner required. The resource pruner can be disabled globally or enabled/disabled on a per resource basis based on an annotation.

## Basic Usage

The following example is a kubebuilder bootstrapped controller that uses basereconciler to reconcile several resources owned by a custom resource. Explanations inline in the code.

```go

}
```