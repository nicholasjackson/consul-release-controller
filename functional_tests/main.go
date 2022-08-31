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
	"path"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/server"
)

var opts = &godog.Options{
	Format: "pretty",
	Output: colors.Colored(os.Stdout),
}

var logStore bytes.Buffer
var shipyardLogStore bytes.Buffer
var logger hclog.Logger
var shipyardLogger hclog.Logger
var releaseServer *server.Release

var environment map[string]string

var createEnvironment = flag.Bool("create-environment", true, "Create and destroy the test environment when running tests?")
var alwaysLog = flag.Bool("always-log", false, "Always show the log output")
var dontDestroy = flag.Bool("dont-destroy", false, "Do not destroy the environment after the scenario")
var dontDestroyError = flag.Bool("dont-destroy-on-error", false, "Do not destroy the environment after an error")

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

var logfile string
var suiteError error

func initializeSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
	})

	ctx.AfterSuite(func() {
		if suiteError != nil {
			fmt.Printf("Error log written to file %s", logfile)

			b, _ := ioutil.ReadFile(logfile)
			fmt.Println(string(b))
		}
	})
}

func initializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^the controller is running on Kubernetes$`, theControllerIsRunningOnKubernetes)
	ctx.Step(`^the controller is running on Nomad$`, theControllerIsRunningOnNomad)

	ctx.Step(`^a Consul "([^"]*)" called "([^"]*)" should be created$`, aConsulCalledShouldBeCreated)
	ctx.Step(`^I create a new Canary "([^"]*)"$`, iCreateANewCanary)

	ctx.Step(`^I create a new version of the Kubernetes deployment "([^"]*)"$`, iDeployANewVersionOfTheKubernetesDeployment)
	ctx.Step(`^I delete the Kubernetes deployment "([^"]*)"$`, iDeleteTheKubernetesDeployment)
	ctx.Step(`^a Kubernetes deployment called "([^"]*)" should exist$`, aKubernetesDeploymentCalledShouldExist)
	ctx.Step(`^a Kubernetes deployment called "([^"]*)" should not exist$`, aKubernetesDeploymentCalledShouldNotExist)
	ctx.Step(`^I create a new Kubernetes release "([^"]*)"$`, iDeployANewVersionOfTheKubernetesRelease)
	ctx.Step(`^I delete the Kubernetes release "([^"]*)"$`, iDeleteTheKubernetesRelease)

	ctx.Step(`^a Nomad job called "([^"]*)" should exist$`, aNomadJobCalledShouldExist)
	ctx.Step(`^a Nomad job called "([^"]*)" should not exist$`, aNomadJobCalledShouldNotExist)
	ctx.Step(`^I create a new Nomad release "([^"]*)"$`, iCreateANewNomadRelease)
	ctx.Step(`^I create a new version of the Nomad job "([^"]*)" called "([^"]*)"$`, iCreateANewVersionOfTheNomadJob)
	ctx.Step(`^I delete the Nomad job "([^"]*)"$`, iDeleteTheNomadJob)
	ctx.Step(`^I delete the Nomad release "([^"]*)"$`, iDeleteTheNomadRelease)

	ctx.Step(`^eventually a call to the URL "([^"]*)" contains the text$`, aCallToTheURLContainsTheText)
	ctx.Step(`^I delete the Canary "([^"]*)"$`, iDeleteTheCanary)

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		environment = map[string]string{}

		if *alwaysLog {
			logger = hclog.New(&hclog.LoggerOptions{Name: "functional-tests", Level: hclog.Trace, Color: hclog.AutoColor})
			shipyardLogger = hclog.New(&hclog.LoggerOptions{Name: "shipyard", Level: hclog.Trace, Color: hclog.AutoColor})
			logger.Info("Create standard logger")
		} else {
			shipyardLogStore = *bytes.NewBufferString("")
			shipyardLogger = hclog.New(&hclog.LoggerOptions{Output: &logStore, Level: hclog.Trace})

			logStore = *bytes.NewBufferString("")
			logger = hclog.New(&hclog.LoggerOptions{Output: &logStore, Level: hclog.Trace})
		}

		return ctx, nil
	})

	ctx.After(func(ctx context.Context, sc *godog.Scenario, scenarioError error) (context.Context, error) {
		logger.Info("Scenario complete, cleanup", "error", scenarioError)

		if scenarioError != nil {
			suiteError = scenarioError
		}

		if releaseServer != nil {
			err := releaseServer.Shutdown()
			if err != nil {
				logger.Error("Unable to shutdown server", "error", err)
				scenarioError = err
			}
		}

		// only destroy the environment when the flag is true
		if *createEnvironment && !*dontDestroy {
			// don't destroy when there is an error and don't destroy error is set
			if scenarioError != nil && *dontDestroyError {
				logger.Info("Don't destroy Shipyard environment")

			} else {
				logger.Info("Destroying Shipyard environment")
				err := executeCommand([]string{"shipyard", "destroy"}, shipyardLogger, false)
				if err != nil {
					logger.Error("Unable to destroy shipyard resources", "error", err)
					scenarioError = err
				}
			}

		}

		for k := range environment {
			os.Unsetenv(k)
		}

		if scenarioError != nil && !*alwaysLog {
			pwd, _ := os.Getwd()
			logfile = path.Join(pwd, "tests.log")

			// create log file
			os.Remove(logfile)
			os.WriteFile(logfile, logStore.Bytes(), os.ModePerm)

		}

		if scenarioError != nil || *dontDestroy {
			// quit all further tests
			return nil, scenarioError
		}

		return ctx, nil
	})
}

func executeCommand(command []string, l hclog.Logger, log bool) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = "../"

	if log {
		cmd.Stdout = l.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug})
		cmd.Stderr = l.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Error})
	}

	return cmd.Run()
}

func startServer() error {
	errChan := make(chan error)

	// set the environment variables
	logger.Debug("Running Server with", "environment", environment)
	for k, v := range environment {
		os.Setenv(k, v)
	}

	var err error
	releaseServer, err = server.New(logger)
	if err != nil {
		logger.Error("Unable to create server", "error", err)
		return err
	}

	go func() {
		err := releaseServer.Start()
		if err != nil {
			logger.Error("Unable to start server", "error", err)
			errChan <- err
		}
	}()

	okChan := time.After(10 * time.Second)
	select {
	case <-okChan:
		return nil
	case err := <-errChan:
		return err
	}
}

func retryOperation(f func() error) error {
	// max time to wait 300s
	attempt := 0
	maxAttempts := 60
	delay := 10 * time.Second

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
