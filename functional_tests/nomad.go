package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

func theControllerIsRunningOnNomad() error {
	// only create the environment when the flag is true

	os.Setenv("LOG_LEVEL", "debug")
	if *createEnvironment {
		err := executeCommand([]string{
			"/usr/local/bin/shipyard",
			"run",
			`--var="install_controller=local"`,
			"./shipyard/nomad",
		}, true)

		if err != nil {
			return fmt.Errorf("unable to create Nomad environment: %s", err)
		}
	}

	// set the shipyard environment variables
	environment["TLS_CERT"] = path.Join(os.Getenv("HOME"), ".shipyard", "data", "nomad_config", "releaser_leaf.cert")
	environment["TLS_KEY"] = path.Join(os.Getenv("HOME"), ".shipyard", "data", "nomad_config", "releaser_leaf.key")
	environment["ENABLE_NOMAD"] = "true"

	// get the variables from shipyard
	output := &strings.Builder{}
	cmd := exec.Command("/usr/local/bin/shipyard", "output")
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
