package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/nicholasjackson/consul-release-controller/pkg/clients"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/sethvargo/go-retry"
)

type validateRequest struct {
	JobHCL       string
	Canonicalize bool
}

func getNomadClient() (clients.Nomad, error) {
	return clients.NewNomad(1*time.Second, 300*time.Second, logger)
}

func theControllerIsRunningOnNomad() error {
	// only create the environment when the flag is true

	os.Setenv("LOG_LEVEL", "debug")
	if *createEnvironment {
		err := executeCommand([]string{
			"shipyard",
			"run",
			`--var="install_controller=local"`,
			"./shipyard/nomad",
		}, shipyardLogger, true)

		if err != nil {
			fmt.Println(shipyardLogStore.String())
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

	cli, _ := getNomadClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := cli.GetHealthyJob(ctx, arg1, "")
	if err != nil {
		return err
	}

	return nil
}

func aNomadJobCalledShouldNotExist(arg1 string) error {
	cli, _ := getNomadClient()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := retry.Constant(ctx, 5*time.Second, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := cli.GetJob(ctx, arg1, "")
		if err != interfaces.ErrDeploymentNotFound {
			return retry.RetryableError(err)
		}

		return nil
	})

	if err == nil {
		return err
	}

	return fmt.Errorf("deployment %s, should not exits", arg1)
}

func iCreateANewNomadRelease(arg1 string) error {
	d, err := ioutil.ReadFile(arg1)
	if err != nil {
		return fmt.Errorf("unable to read file %s: %s", arg1, err)
	}

	// validate the config with the Nomad API
	r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/releases", "https://localhost:9443"), bytes.NewReader(d))
	if err != nil {
		return fmt.Errorf("unable to create http request: %s", err)
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.DefaultClient.Do(r)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error parsing creating release, expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func iCreateANewVersionOfTheNomadJob(arg1, arg2 string) error {
	// load the file
	d, err := ioutil.ReadFile(arg1)
	if err != nil {
		return fmt.Errorf("unable to read file %s: %s", arg1, err)
	}

	// build the request object
	rd := validateRequest{
		JobHCL: string(d),
	}

	jobData, _ := json.Marshal(rd)

	// validate the config with the Nomad API

	r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/jobs/parse", environment["NOMAD_ADDR"]), bytes.NewReader(jobData))
	if err != nil {
		return fmt.Errorf("unable to create http request: %s", err)
	}

	resp, err := http.DefaultClient.Do(r)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error parsing job data, expected status 200, got %d", resp.StatusCode)
	}

	jobResp, _ := ioutil.ReadAll(resp.Body)
	logger.Info("data json", "job", string(jobResp))

	job := &api.Job{}
	job.ID = &arg2

	err = json.Unmarshal(jobResp, job)
	if err != nil {
		return fmt.Errorf("unable to marshal job data: %s", err)
	}

	if job.Meta == nil {
		job.Meta = map[string]string{}
	}

	job.Meta["updated"] = time.Now().String()

	logger.Info("data", "job", job)

	cli, _ := getNomadClient()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return cli.UpsertJob(ctx, job)
}

func iDeleteTheNomadJob(arg1 string) error {
	cli, _ := getNomadClient()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return cli.DeleteJob(ctx, arg1, "")
}

func iDeleteTheNomadRelease(arg1 string) error {
	// validate the config with the Nomad API
	r, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v1/releases/%s", "https://localhost:9443", arg1), nil)
	if err != nil {
		return fmt.Errorf("unable to create http request: %s", err)
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return fmt.Errorf("unable to delete release: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to delete release, expected status 200, got %d", resp.StatusCode)
	}

	return nil
}
