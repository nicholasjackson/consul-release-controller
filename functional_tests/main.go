package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/hashicorp/go-hclog"
)

var opts = &godog.Options{
	Format: "pretty",
	Output: colors.Colored(os.Stdout),
}

var logStore bytes.Buffer
var logger hclog.Logger

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
		logStore = *bytes.NewBufferString("")
		logger = hclog.New(&hclog.LoggerOptions{Output: &logStore})
	})
}

func initializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^the controller is running on Kubernetes$`, theControllerIsRunningOnKubernetes)

	ctx.After(func(ctx context.Context, sc *godog.Scenario, scenarioError error) (context.Context, error) {
		showLog := false
		if scenarioError != nil {
			showLog = true
		}

		cmd := exec.Command("/usr/local/bin/shipyard", "destroy")
		cmd.Dir = "../"
		cmd.Stdout = logger.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug})
		cmd.Stderr = logger.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Error})

		err := cmd.Run()
		if err != nil {
			showLog = true
		}

		if showLog {
			fmt.Println(logStore.String())
		}
		return ctx, err
	})
}

func theControllerIsRunningOnKubernetes() error {
	cmd := exec.Command("/usr/local/bin/shipyard", "run", "./shipyard/kubernetes")
	cmd.Dir = "../"
	cmd.Stdout = logger.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug})
	cmd.Stderr = logger.StandardWriter(&hclog.StandardLoggerOptions{ForceLevel: hclog.Error})

	err := cmd.Run()
	return err
}
