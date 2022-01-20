package clients

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	v1release "github.com/nicholasjackson/consul-release-controller/kubernetes/controller/api/v1"
	"github.com/sethvargo/go-retry"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controller "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DeploymentNotFound   = "deployment_not_found"
	DeploymentNotHealthy = "deployment_not_healthy"
)

var ErrDeploymentNotFound = fmt.Errorf(DeploymentNotFound)
var ErrDeploymentNotHealthy = fmt.Errorf(DeploymentNotHealthy)

// Kubernetes is a high level functional interface for interacting with the Kubernetes API
type Kubernetes interface {
	// GetDeployment returns a Kubernetes deployment matching the given name and
	// namespace.
	// If the deployment does not exist a DeploymentNotFound error will be returned
	// and a nil deployments
	// Any other error than DeploymentNotFound can be treated like an internal error
	// in executing the request
	GetDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error)

	// UpsertDeployment creates or updates the given Kubernetes Deployment
	UpsertDeployment(ctx context.Context, dep *appsv1.Deployment) error

	// DeleteDeployment deletes the given Kubernetes Deployment
	DeleteDeployment(ctx context.Context, name, namespace string) error

	// GetHealthyDeployment blocks until a healthy deployment is found or the process times out
	// returns the Deployment an a nil error on success
	// returns a nil deployment and a ErrDeploymentNotFound error when the deployment does not exist
	// returns a nill deployment and a ErrDeploymentNotHealthy error when the deployment exists but is not in a healthy state
	// any other error type signifies an internal error
	GetHealthyDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error)

	// InsertRelease creates or updates the given Kubernetes Release
	InsertRelease(ctx context.Context, dep *v1release.Release) error

	// DeleteRelease deletes the given Kubernetes Release
	DeleteRelease(ctx context.Context, name, namespace string) error
}

// NewKubernetes creates a new Kubernetes implementation
func NewKubernetes(configPath string, timeout, interval time.Duration, l hclog.Logger) (Kubernetes, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, fmt.Errorf("unable to build Kubernetes config using path: %s, error: %s", configPath, err)
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create Kubernetes client, error: %s", err)
	}

	scheme := runtime.NewScheme()
	v1release.AddToScheme(scheme)

	cc, err := controller.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("unable to create controller client, error: %s", err)
	}

	return &KubernetesImpl{clientset: cs, controllerClient: cc, timeout: timeout, interval: interval, logger: l}, nil
}

type KubernetesImpl struct {
	clientset        *kubernetes.Clientset
	controllerClient controller.Client
	timeout          time.Duration
	interval         time.Duration
	logger           hclog.Logger
}

func (k *KubernetesImpl) GetDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
	d, err := k.clientset.AppsV1().Deployments(namespace).Get(ctx, name, v1.GetOptions{})

	if errors.IsNotFound(err) {
		return nil, ErrDeploymentNotFound
	}

	return d, err
}

func (k *KubernetesImpl) UpsertDeployment(ctx context.Context, dep *appsv1.Deployment) error {
	_, err := k.GetDeployment(ctx, dep.Name, dep.Namespace)
	if err == ErrDeploymentNotFound {
		_, err = k.clientset.AppsV1().Deployments(dep.Namespace).Create(ctx, dep, v1.CreateOptions{})
		return err
	}

	if err != nil {
		return err
	}

	_, err = k.clientset.AppsV1().Deployments(dep.Namespace).Update(ctx, dep, v1.UpdateOptions{})
	return err
}

func (k *KubernetesImpl) DeleteDeployment(ctx context.Context, name, namespace string) error {
	thirty := int64(30)
	err := k.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, v1.DeleteOptions{GracePeriodSeconds: &thirty})

	if errors.IsNotFound(err) {
		return ErrDeploymentNotFound
	}

	return err
}

// getHealthyDeployment gets the named kubernetes deployment and blocks until it is healthy
func (k *KubernetesImpl) GetHealthyDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
	retryContext, cancel := context.WithTimeout(ctx, k.timeout)
	defer cancel()

	var deployment *appsv1.Deployment
	var lastError error

	err := retry.Constant(retryContext, k.interval, func(ctx context.Context) error {
		k.logger.Debug("Checking health", "name", name, "namespace", namespace)

		deployment, lastError = k.GetDeployment(ctx, name, namespace)
		if lastError == ErrDeploymentNotFound {
			k.logger.Debug("Deployment not found", "name", name, "namespace", namespace, "error", lastError)

			return retry.RetryableError(lastError)
		}

		if lastError != nil {
			k.logger.Error("Unable to call GetDeployment", "name", name, "namespace", namespace, "error", lastError)

			return retry.RetryableError(fmt.Errorf("error calling GetDeployment: %s", lastError))
		}

		k.logger.Debug(
			"Deployment health",
			"name", name,
			"namespace", namespace,
			"status_replicas", deployment.Status.AvailableReplicas,
			"desired_replicas", deployment.Status.Replicas)

		if deployment.Status.UnavailableReplicas > 0 || deployment.Status.AvailableReplicas < 1 {
			k.logger.Debug("Deployment not healthy", "name", name, "namespace", namespace)
			lastError = ErrDeploymentNotHealthy

			return retry.RetryableError(ErrDeploymentNotHealthy)
		}

		k.logger.Debug("Deployment healthy", "name", name, "namespace", namespace)

		return nil
	})

	if os.IsTimeout(err) {
		k.logger.Error("Timeout waiting for healthy deployment", "name", name, "namespace", namespace, "error", lastError)

		return nil, lastError
	}

	return deployment, nil
}

func (k *KubernetesImpl) InsertRelease(ctx context.Context, release *v1release.Release) error {
	err := k.controllerClient.Create(ctx, release)

	return err
}

func (k *KubernetesImpl) DeleteRelease(ctx context.Context, name, namespace string) error {
	obj := v1release.Release{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: namespace},
	}

	thirty := int64(30)
	err := k.controllerClient.Delete(ctx, &obj, &controller.DeleteOptions{GracePeriodSeconds: &thirty})

	if errors.IsNotFound(err) {
		return ErrDeploymentNotFound
	}

	return err
}
