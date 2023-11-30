package reconciler

type ReconcilerConfig struct {
	AnnotationsDomain string
	ResourcePruner    bool
}

var Config ReconcilerConfig = ReconcilerConfig{
	AnnotationsDomain: "basereconciler.3cale.net",
	ResourcePruner:    true,
}
