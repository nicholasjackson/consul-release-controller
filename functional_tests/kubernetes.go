package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nicholasjackson/consul-release-controller/pkg/clients"
	v1release "github.com/nicholasjackson/consul-release-controller/pkg/controllers/kubernetes/api/v1"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/sethvargo/go-retry"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var timeout = 30 * time.Second

func getKubernetesClient() (clients.Kubernetes, error) {
	return clients.NewKubernetes(os.Getenv("KUBECONFIG"), 120*time.Second, 1*time.Second, logger.Named("kubernetes-client"))
}

func theControllerIsRunningOnKubernetes() error {
	// only create the environment when the flag is true

	os.Setenv("LOG_LEVEL", "debug")
	if *createEnvironment {
		logger.Info("Creating Shipyard environment")

		err := executeCommand([]string{
			"shipyard",
			"run",
			`--var="helm_controller_enabled=false"`,
			"./shipyard/kubernetes",
		}, shipyardLogger, true)

		if err != nil {
			fmt.Println(shipyardLogStore.String())
			return fmt.Errorf("unable to create Kubernetes environment: %s", err)
		}
	}

	// set the shipyard environment variables
	environment["ENABLE_KUBERNETES"] = "true"

	// get the variables from shipyard
	output := &strings.Builder{}
	cmd := exec.Command("shipyard", "output")
	cmd.Dir = "../"
	cmd.Stdout = output
	cmd.Stderr = output

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("unable to get output variables from shipyard: %s", err)
	}

	shipyardOutput := map[string]string{}
	err = json.Unmarshal([]byte(output.String()), &shipyardOutput)
	if err != nil {
		return fmt.Errorf("unable to parse shipyard output: %s", err)
	}

	for k, v := range shipyardOutput {
		environment[k] = v
	}

	return startServer()
}

func iDeployANewVersionOfTheKubernetesDeployment(arg1 string) error {
	d, err := ioutil.ReadFile(arg1)
	if err != nil {
		return fmt.Errorf("unable to read Kubernetes deployment: %s", err)
	}

	dep := &appsv1.Deployment{}
	err = yaml.Unmarshal(d, dep)
	if err != nil {
		return fmt.Errorf("unable to decode Kubernetes deployment: %s", err)
	}

	// force the update
	if dep.Annotations == nil {
		dep.Annotations = map[string]string{}
	}

	dep.Annotations["updated"] = time.Now().String()

	cs, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("unable to create Kubernetes client, error: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err = cs.UpsertKubernetesDeployment(ctx, dep)
	if err != nil {
		return fmt.Errorf("unable to create Kubernetes deployment, error: %s", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err = cs.GetHealthyDeployment(ctx, dep.Name, dep.Namespace)
	if err != nil {
		logger.Debug("Kubernetes deployment not found", "name", dep.Name, "namespace", dep.Namespace, "error", err)
		return fmt.Errorf("unable to find deployment: %s, %s", dep.Name, err)
	}

	return nil
}

func iDeleteTheKubernetesDeployment(name string) error {
	cs, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("unable to create Kubernetes client, error: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = cs.DeleteDeployment(ctx, name, "default")
	if err == nil {
		return nil
	}

	if err != interfaces.ErrDeploymentNotFound {
		return fmt.Errorf("unable to delete deployment: %s, %s", name, err)
	}

	return nil
}

func aKubernetesDeploymentCalledShouldExist(arg1 string) error {
	cs, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("unable to create Kubernetes client, error: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err = cs.GetHealthyKubernetesDeployment(ctx, arg1, "default")
	if err != nil {
		return fmt.Errorf("unable to get Kubernetes deployment: %s, error: %s", arg1, err)
	}

	return err
}

func aKubernetesDeploymentCalledShouldNotExist(arg1 string) error {
	cs, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("unable to create Kubernetes client, error: %s", err)
	}

	retryContext, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = retry.Fibonacci(retryContext, 1*time.Second, func(ctx context.Context) error {

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		dep, err := cs.GetKubernetesDeployment(ctx, arg1, "default")
		if err == interfaces.ErrDeploymentNotFound {
			return nil
		}

		// if the deployment exists check to see if it has been scaled to 0
		if err == nil {
			if dep.Status.AvailableReplicas == 0 {
				return nil
			}
		}

		return retry.RetryableError(fmt.Errorf("kubernetes deployment should not exist, or should have scale 0"))
	})

	return err
}

func iDeployANewVersionOfTheKubernetesRelease(arg1 string) error {
	d, err := ioutil.ReadFile(arg1)
	if err != nil {
		return fmt.Errorf("unable to read Kubernetes release: %s", err)
	}

	rel := &v1release.Release{}
	err = yaml.Unmarshal(d, rel)
	if err != nil {
		return fmt.Errorf("unable to decode Kuberneteh release: %s", err)
	}

	// force the update
	if rel.Annotations == nil {
		rel.Annotations = map[string]string{}
	}

	rel.Annotations["updated"] = time.Now().String()

	cs, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("unable to create Kubernetes client, error: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = cs.InsertRelease(ctx, rel)
	if err != nil {
		return fmt.Errorf("unable to create Kubernetes release, error: %s", err)
	}

	return nil
}

func iDeleteTheKubernetesRelease(name string) error {
	cs, err := getKubernetesClient()
	if err != nil {
		return fmt.Errorf("unable to create Kubernetes client, error: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = cs.DeleteRelease(ctx, name, "default")
	if err == nil {
		return nil
	}

	if err != interfaces.ErrDeploymentNotFound {
		return fmt.Errorf("unable to delete release: %s", err)
	}

	return nil
}
