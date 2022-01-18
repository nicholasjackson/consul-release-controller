package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/controller"
)

var opts = &godog.Options{
	Format: "pretty",
	Output: colors.Colored(os.Stdout),
}

var logStore bytes.Buffer
var logger hclog.Logger
var server *controller.Release

var environment map[string]string

var createEnvironment = flag.Bool("create-environment", true, "Create and destroy the test environment when running tests?")

func main() {
	godog.BindFlags("godog.", flag.CommandLine, opts)
	flag.Parse()

	status := godog.TestSuite{
		Name:                 "Controller Functional Tests",
		ScenarioInitializer:  initializeScenario,
		TestSuiteInitializer: initializeSuite,
		Options:              opts,
	}.Run()

	os.Exit(status)
}

func initializeSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		environment = map[string]string{}

		logStore = *bytes.NewBufferString("")
		logger = hclog.New(&hclog.LoggerOptions{Output: &logStore})
	})
}

func initializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^the controller is running on Kubernetes$`, theControllerIsRunningOnKubernetes)
	ctx.Step(`^a Consul "([^"]*)" called "([^"]*)" should be created$`, aConsulCalledShouldBeCreated)
	ctx.Step(`^I create a new Canary "([^"]*)"$`, iCreateANewCanary)
	ctx.Step(`^I create a new version of the Kubernetes Deployment "([^"]*)"$`, iDeployANewVersionOfTheKubernetesDeployment)

	ctx.Step(`^a Kubernetes deployment called "([^"]*)" should exist$`, aKubernetesDeploymentCalledShouldExist)
	ctx.Step(`^a Kubernetes deployment called "([^"]*)" should not exist$`, aKubernetesDeploymentCalledShouldNotExist)
	ctx.Step(`^eventually a call to the URL "([^"]*)" contains the text$`, aCallToTheURLContainsTheText)
	ctx.Step(`^I delete the Canary "([^"]*)"$`, iDeleteTheCanary)

	ctx.After(func(ctx context.Context, sc *godog.Scenario, scenarioError error) (context.Context, error) {
		showLog := false
		if scenarioError != nil {
			showLog = true
		}

		if server != nil {
			err := server.Shutdown()
			if err != nil {
				showLog = true
			}
		}

		// only destroy the environment when the flag is true
		if *createEnvironment {
			err := executeCommand([]string{"/usr/local/bin/shipyard", "destroy"})
			if err != nil {
				showLog = true
			}
		}

		if showLog {
			fmt.Println(logStore.String())
		}

		return ctx, nil
	})
}

func executeCommand(command []string) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = "../"
	cmd.Stdout = logger.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug})
	cmd.Stderr = logger.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Error})

	return cmd.Run()
}

func startServer() error {
	var err error

	// set the environment variables
	for k, v := range environment {
		os.Setenv(k, v)
	}

	server = controller.New()

	go func() {
		err = server.Start()
	}()

	// wait for the server to start and return any error
	time.Sleep(5 * time.Second)
	return err
}

func retryOperation(f func() error) error {
	// max time to wait 300s
	attempt := 0
	maxAttempts := 60
	delay := 5 * time.Second

	var funcError error
	for attempt = 0; attempt < maxAttempts; attempt++ {
		funcError = f()
		if funcError == nil {
			return nil
		}

		time.Sleep(delay)
	}

	return funcError
}

func aConsulCalledShouldBeCreated(arg1, arg2 string) error {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return fmt.Errorf("unable to create Consul client: %s", err)
	}

	err = retryOperation(func() error {
		_, _, err := client.ConfigEntries().Get(arg1, arg2, nil)
		return err
	})

	return err
}

func iCreateANewCanary(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("unable to open canary file: %s", err)
	}
	defer f.Close()

	req, err := http.NewRequest(http.MethodPost, "https://localhost:9443/v1/releases", f)
	if err != nil {
		return fmt.Errorf("unable to create request: %s", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to write config: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to write config: expected code 200, got %d", resp.StatusCode)
	}

	return nil
}

func iDeleteTheCanary(name string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("https://localhost:9443/v1/releases/%s", name), nil)
	if err != nil {
		return fmt.Errorf("unable to create request: %s", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to write config: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to write config: expected code 200, got %d", resp.StatusCode)
	}

	return nil
}

func aCallToTheURLContainsTheText(addr string, text *godog.DocString) error {
	return retryOperation(func() error {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		r, err := http.Get(addr)
		if err != nil {
			return err
		}
		defer r.Body.Close()

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}

		if !strings.Contains(string(b), text.Content) {
			return fmt.Errorf("request body: %s does not contain: %s", string(b), text.Content)
		}

		return nil
	})
}
