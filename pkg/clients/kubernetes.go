package clients

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	v1release "github.com/nicholasjackson/consul-release-controller/pkg/controllers/kubernetes/api/v1"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/sethvargo/go-retry"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controller "sigs.k8s.io/controller-runtime/pkg/client"
)

// Kubernetes defines an interface for a Kubernetes client
type Kubernetes interface {
	interfaces.RuntimeClient

	// GetKubernetesDeployment returns a appsv1.Deployment for the given parameters using a regex to match the deployment name
	GetKubernetesDeploymentWithSelector(ctx context.Context, selector, namespace string) (*appsv1.Deployment, error)

	// GetKubernetesDeployment returns an appsv1.Deployment for the given name and namespace
	GetKubernetesDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error)

	// GetHealthyKubernetes deployment finds a deployment and returns an appsv1.Deployment only when the deployment is healthy
	GetHealthyKubernetesDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error)

	// UpsertKubernetesDeployment creates or updates the given Kubernetes Deployment
	UpsertKubernetesDeployment(ctx context.Context, dep *appsv1.Deployment) error

	// InsertRelease creates or updates the given Kubernetes Release
	InsertRelease(ctx context.Context, dep *v1release.Release) error

	// DeleteRelease deletes the given Kubernetes Release
	DeleteRelease(ctx context.Context, name, namespace string) error
}

// NewKubernetes creates a new Kubernetes implementation
func NewKubernetes(configPath string, timeout, interval time.Duration, l hclog.Logger) (Kubernetes, error) {
	var err error
	var config *rest.Config

	if configPath == "" {
		// assume we are running in a cluster and have the correct permissions on the pod
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to create Kubernetes config from in cluster config, please check the controller has the correct permissions")
		}
	} else {
		// if whe have an explicit config file path, build the config from that
		config, err = clientcmd.BuildConfigFromFlags("", configPath)
		if err != nil {
			return nil, fmt.Errorf("unable to build Kubernetes config using path: %s, error: %s", configPath, err)
		}
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

// KubernetesImpl is the concrete implementation of the Kubernetes client interface
type KubernetesImpl struct {
	clientset        *kubernetes.Clientset
	controllerClient controller.Client
	timeout          time.Duration
	interval         time.Duration
	logger           hclog.Logger
}

func (k *KubernetesImpl) GetKubernetesDeploymentWithSelector(ctx context.Context, selector, namespace string) (*appsv1.Deployment, error) {
	deps, err := k.clientset.AppsV1().Deployments(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, interfaces.ErrDeploymentNotFound
	}

	if !strings.HasSuffix(selector, "$") {
		selector = selector + "$"
	}

	re, err := regexp.Compile(selector)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression for deployment selector: %s, error: %s", selector, err)
	}

	// iterate over the list and look for a match
	for _, d := range deps.Items {
		if re.MatchString(d.Name) {
			return k.GetKubernetesDeployment(ctx, d.Name, namespace)
		}
	}

	return nil, interfaces.ErrDeploymentNotFound
}

func (k *KubernetesImpl) GetKubernetesDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
	d, err := k.clientset.AppsV1().Deployments(namespace).Get(ctx, name, v1.GetOptions{})

	if errors.IsNotFound(err) {
		return nil, interfaces.ErrDeploymentNotFound
	}

	return d, err
}

func (k *KubernetesImpl) UpsertKubernetesDeployment(ctx context.Context, dep *appsv1.Deployment) error {
	// set modified by
	if dep.Labels == nil {
		dep.Labels = map[string]string{}
	}

	dep.Labels["consul-release-controller-version"] = dep.ResourceVersion

	_, err := k.GetDeployment(ctx, dep.Name, dep.Namespace)
	if err == interfaces.ErrDeploymentNotFound {
		_, err = k.clientset.AppsV1().Deployments(dep.Namespace).Create(ctx, dep, v1.CreateOptions{})
		return err
	}

	if err != nil {
		return err
	}

	_, err = k.clientset.AppsV1().Deployments(dep.Namespace).Update(ctx, dep, v1.UpdateOptions{})
	return err
}

func (k *KubernetesImpl) DeleteKubernetesDeployment(ctx context.Context, name, namespace string) error {
	thirty := int64(30)
	err := k.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, v1.DeleteOptions{GracePeriodSeconds: &thirty})

	if errors.IsNotFound(err) {
		return interfaces.ErrDeploymentNotFound
	}

	return err
}

// getHealthyDeployment gets the named kubernetes deployment and blocks until it is healthy
func (k *KubernetesImpl) GetHealthyKubernetesDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
	retryContext, cancel := context.WithTimeout(ctx, k.timeout)
	defer cancel()

	var deployment *appsv1.Deployment
	var lastError error

	err := retry.Constant(retryContext, k.interval, func(ctx context.Context) error {
		k.logger.Debug("Checking health", "name", name, "namespace", namespace)

		deployment, lastError = k.GetKubernetesDeployment(ctx, name, namespace)
		if lastError == interfaces.ErrDeploymentNotFound {
			k.logger.Debug("Deployment not found", "name", name, "namespace", namespace, "error", lastError)

			return retry.RetryableError(lastError)
		}

		if lastError != nil {
			k.logger.Error("Unable to call GetDeployment", "name", name, "namespace", namespace, "error", lastError)

			return retry.RetryableError(fmt.Errorf("error calling GetDeployment: %s", lastError))
		}

		zero := int32(0)
		// if the scale is set to 0 fail fast and return deployment not found
		if deployment.Spec.Replicas == &zero {
			return interfaces.ErrDeploymentNotFound
		}

		k.logger.Debug(
			"Deployment health",
			"name", name,
			"namespace", namespace,
			"status_replicas", deployment.Status.AvailableReplicas,
			"desired_replicas", deployment.Status.Replicas)

		if deployment.Status.UnavailableReplicas > 0 || deployment.Status.AvailableReplicas < 1 {
			k.logger.Debug("Deployment not healthy", "name", name, "namespace", namespace)
			lastError = interfaces.ErrDeploymentNotHealthy

			return retry.RetryableError(interfaces.ErrDeploymentNotHealthy)
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
		return interfaces.ErrDeploymentNotFound
	}

	return err
}

func (k *KubernetesImpl) GetDeployment(ctx context.Context, name, namespace string) (*interfaces.Deployment, error) {
	dep, err := k.GetKubernetesDeployment(ctx, name, namespace)
	if dep != nil {
		d := &interfaces.Deployment{
			Name:            dep.Name,
			Namespace:       dep.Namespace,
			Meta:            dep.Labels,
			Instances:       int(*dep.Spec.Replicas),
			ResourceVersion: dep.ResourceVersion,
		}

		return d, err
	}

	return nil, err
}

func (k *KubernetesImpl) GetDeploymentWithSelector(ctx context.Context, selector, namespace string) (*interfaces.Deployment, error) {
	dep, err := k.GetKubernetesDeploymentWithSelector(ctx, selector, namespace)
	if dep != nil {
		d := &interfaces.Deployment{
			Name:            dep.Name,
			Namespace:       dep.Namespace,
			Meta:            dep.Labels,
			Instances:       int(*dep.Spec.Replicas),
			ResourceVersion: dep.ResourceVersion,
		}

		return d, err
	}

	return nil, err
}

func (k *KubernetesImpl) UpdateDeployment(ctx context.Context, deployment *interfaces.Deployment) error {
	dep, err := k.GetKubernetesDeployment(ctx, deployment.Name, deployment.Namespace)
	if err != nil {
		return err
	}

	replicas := int32(deployment.Instances)
	dep.Labels = deployment.Meta
	dep.ResourceVersion = deployment.ResourceVersion
	dep.Spec.Replicas = &replicas

	return k.UpsertKubernetesDeployment(ctx, dep)
}

func (k *KubernetesImpl) CloneDeployment(ctx context.Context, existingDeployment *interfaces.Deployment, newDeployment *interfaces.Deployment) error {
	dep, err := k.GetKubernetesDeployment(ctx, existingDeployment.Name, existingDeployment.Namespace)
	if err != nil {
		return err
	}

	clone := dep.DeepCopy()

	clone.Name = newDeployment.Name
	clone.Namespace = newDeployment.Namespace
	clone.Labels = newDeployment.Meta
	clone.ResourceVersion = newDeployment.ResourceVersion

	return k.UpsertKubernetesDeployment(ctx, clone)
}

func (k *KubernetesImpl) DeleteDeployment(ctx context.Context, name, namespace string) error {
	return k.DeleteKubernetesDeployment(ctx, name, namespace)
}

func (k *KubernetesImpl) GetHealthyDeployment(ctx context.Context, name, namespace string) (*interfaces.Deployment, error) {
	dep, err := k.GetHealthyKubernetesDeployment(ctx, name, namespace)
	if err != nil {
		return nil, err
	}

	d := &interfaces.Deployment{
		Name:            dep.Name,
		Namespace:       dep.Namespace,
		Meta:            dep.Labels,
		Instances:       int(*dep.Spec.Replicas),
		ResourceVersion: dep.ResourceVersion,
	}

	return d, nil
}

// CandidateSubsetFilter retrurns the Consul resolver subset filter that should be used for this runtime to identify candidate instances
func (k *KubernetesImpl) CandidateSubsetFilter() string {
	return fmt.Sprintf(`Service.ID not contains "%s"`, "primary")
}

// PrimarySubsetFilter returns the Consul resolver subset filter that should be used for this runtime to identify the primary instances
func (k *KubernetesImpl) PrimarySubsetFilter() string {
	return fmt.Sprintf(`Service.ID contains "%s"`, "primary")
}
