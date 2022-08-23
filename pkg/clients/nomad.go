package clients

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/sethvargo/go-retry"
)

type Nomad interface {
	// GetJob returns a Nomad job matching the given name and
	// namespace.
	// If the job does not exist a DeploymentNotFound error will be returned
	// and a nil job
	// Any other error than DeploymentNotFound can be treated like an internal error
	// in executing the request
	GetJob(ctx context.Context, name, namespace string) (*api.Job, error)

	GetHealthyJob(ctx context.Context, name, namespace string) (*api.Job, error)

	GetJobWithSelector(ctx context.Context, selector, namespace string) (*api.Job, error)

	UpsertJob(ctx context.Context, job *api.Job) error

	DeleteJob(ctx context.Context, id string, namespace string) error

	GetEvents(ctx context.Context) (<-chan *api.Events, error)
}

type NomadImpl struct {
	client   *api.Client
	log      hclog.Logger
	interval time.Duration
	timeout  time.Duration
}

func NewNomad(interval, timeout time.Duration, l hclog.Logger) (Nomad, error) {
	conf := api.DefaultConfig()
	c, err := api.NewClient(conf)
	if err != nil {
		return nil, fmt.Errorf("unable to create Nomad client: %s", err)
	}

	return &NomadImpl{client: c, log: l, interval: interval, timeout: timeout}, nil
}

func (ni *NomadImpl) GetJobWithSelector(ctx context.Context, selector, namespace string) (*api.Job, error) {
	qo := &api.QueryOptions{
		Namespace: namespace,
	}

	jobs, _, err := ni.client.Jobs().List(qo)
	if err != nil {
		return nil, fmt.Errorf("unable to list jobs: %s", err)
	}

	if !strings.HasSuffix(selector, "$") {
		selector = selector + "$"
	}

	re, err := regexp.Compile(selector)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression for deployment selector: %s, error: %s", selector, err)
	}

	for _, j := range jobs {
		if re.MatchString(j.Name) {
			j, _, err := ni.client.Jobs().Info(j.ID, &api.QueryOptions{})
			if err != nil {
				return nil, err
			}

			status := j.Status
			if status != nil && (*status == "running" || *status == "pending") {
				return j, nil
			}
		}
	}

	return nil, ErrDeploymentNotFound
}

func (ni *NomadImpl) GetJob(ctx context.Context, name, namespace string) (*api.Job, error) {

	return ni.GetJobWithSelector(ctx, name, namespace)
}

func (ni *NomadImpl) UpsertJob(ctx context.Context, job *api.Job) error {
	wo := &api.WriteOptions{}

	_, _, err := ni.client.Jobs().Register(job, wo)
	return err
}

func (ni *NomadImpl) DeleteJob(ctx context.Context, id, namespace string) error {
	_, err := ni.GetJob(ctx, id, namespace)
	if err != nil {
		return err
	}

	_, _, err = ni.client.Jobs().Deregister(id, true, &api.WriteOptions{Namespace: namespace})
	return err
}

func (ni *NomadImpl) GetHealthyJob(ctx context.Context, name, namespace string) (*api.Job, error) {
	retryContext, cancel := context.WithTimeout(ctx, ni.timeout)
	defer cancel()

	var job *api.Job
	var lastError error

	err := retry.Constant(retryContext, ni.interval, func(ctx context.Context) error {
		ni.log.Debug("Checking health", "name", name, "namespace", namespace)

		job, lastError = ni.GetJob(ctx, name, namespace)

		if lastError == ErrDeploymentNotFound {
			ni.log.Debug("Job not found", "name", name, "namespace", namespace, "error", lastError)

			return retry.RetryableError(lastError)
		}

		if lastError != nil {
			ni.log.Error("Unable to call GetJob", "name", name, "namespace", namespace, "error", lastError)

			return retry.RetryableError(fmt.Errorf("error calling GetJob: %s", lastError))
		}

		// get the allocations
		allocs, _, lastError := ni.client.Jobs().Allocations(*job.ID, true, &api.QueryOptions{Namespace: namespace})
		if lastError != nil {
			ni.log.Error("Unable to call GetAllocations", "name", name, "namespace", namespace, "error", lastError)

			return retry.RetryableError(fmt.Errorf("error calling GetJob: %s", lastError))
		}

		desired := 0
		healthy := 0

		for _, a := range allocs {
			if a.DesiredStatus == "run" {
				desired++

				if a.ClientStatus == "running" {
					healthy++
				}
			}
		}

		ni.log.Debug(
			"Job health",
			"name", name,
			"namespace", namespace,
			"desired", desired,
			"healthy", healthy)

		if healthy == 0 || desired != healthy {
			return retry.RetryableError(fmt.Errorf("%d of %d allocations healthy, retry", healthy, desired))
		}

		return nil
	})

	if os.IsTimeout(err) {
		ni.log.Error("Timeout waiting for healthy job", "name", name, "namespace", namespace, "error", lastError)

		return nil, lastError
	}

	return job, nil
}

func (ni *NomadImpl) GetEvents(ctx context.Context) (<-chan *api.Events, error) {

	topics := map[api.Topic][]string{
		api.TopicJob: []string{"*"},
	}

	return ni.client.EventStream().Stream(ctx, topics, 9999999, &api.QueryOptions{})
}
