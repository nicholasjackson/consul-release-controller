package platforms

// Runtime defines an interface that all concrete platforms like Kubernetes must
// implement
type Runtime interface {
	// Deploy the new test version to the platform
	Deploy() error
	// Rollback the test version
	Rollback() error
}
