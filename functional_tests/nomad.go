package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cucumber/godog"
)

func theControllerIsRunningOnNomad() error {
	// only create the environment when the flag is true

	os.Setenv("LOG_LEVEL", "debug")
	if *createEnvironment {
		err := executeCommand([]string{
			"shipyard",
			"run",
			`--var="install_controller=local"`,
			"./shipyard/nomad",
		}, true)

		if err != nil {
			return fmt.Errorf("unable to create Nomad environment: %s", err)
		}
	}

	// set the shipyard environment variables
	environment["ENABLE_NOMAD"] = "true"

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

func aNomadJobCalledShouldExist(arg1 string) error {
	return godog.ErrPending
}

func aNomadJobCalledShouldNotExist(arg1 string) error {
	return godog.ErrPending
}

func iCreateANewNomadRelease(arg1 string) error {
	return godog.ErrPending
}

func iCreateANewVersionOfTheNomadJob(arg1 string) error {
	return godog.ErrPending
}

func iDeleteTheNomadJob(arg1 string) error {
	return godog.ErrPending

}

func iDeleteTheNomadRelease(arg1 string) error {
	return godog.ErrPending
}

func nomadJobCalledShouldExist(arg1 string) error {
	return godog.ErrPending
}
