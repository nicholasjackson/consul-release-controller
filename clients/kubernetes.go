package clients

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	DeploymentNotFound = "deployment_not_found"
)

// Kubernetes is a high level functional interface for interacting with the Kubernetes API
type Kubernetes interface {
	// GetDeployment returns a Kubernetes deployment matching the given name and
	// namespace.
	// If the deployment does not exist a DeploymentNotFound error will be returned
	// and a nil deployments
	// Any other error than DeploymentNotFound can be treated like an internal error
	// in executing the request
	GetDeployment(name, namespace string) (*appsv1.Deployment, error)

	// UpsertDeployment creates or updates the given Kubernetes Deployment
	UpsertDeployment(d *appsv1.Deployment) error
}

// NewKubernetes creates a new Kubernetes implementation
func NewKubernetes(configPath string) (Kubernetes, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, fmt.Errorf("unable to build Kubernetes config using path: %s, error: %s", configPath, err)
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create Kubernetes client, error: %s", err)
	}

	return &KubernetesImpl{clientset: cs}, nil
}

type KubernetesImpl struct {
	clientset *kubernetes.Clientset
}

func (k *KubernetesImpl) GetDeployment(name, namespace string) (*appsv1.Deployment, error) {
	return k.clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, v1.GetOptions{})
}

func (k *KubernetesImpl) UpsertDeployment(dep *appsv1.Deployment) error {
	if d, err := k.clientset.AppsV1().Deployments(dep.Namespace).Get(context.Background(), dep.Name, v1.GetOptions{}); err == nil && d != nil {
		_, err := k.clientset.AppsV1().Deployments(dep.Namespace).Update(context.Background(), dep, v1.UpdateOptions{})
		return err
	}

	_, err := k.clientset.AppsV1().Deployments(dep.Namespace).Create(context.Background(), dep, v1.CreateOptions{})
	return err
}
