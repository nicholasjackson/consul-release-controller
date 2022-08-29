package clients

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/sethvargo/go-retry"
)

type Nomad interface {
	interfaces.RuntimeClient

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

	return nil, interfaces.ErrDeploymentNotFound
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

		if lastError == interfaces.ErrDeploymentNotFound {
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

// GetDeployment returns a Kubernetes deployment matching the given name and
// namespace.
// If the deployment does not exist a DeploymentNotFound error will be returned
// and a nil deployments
// Any other error than DeploymentNotFound can be treated like an internal error
// in executing the request
func (ni *NomadImpl) GetDeployment(ctx context.Context, name, namespace string) (*interfaces.Deployment, error) {
	job, err := ni.GetJob(ctx, name, namespace)

	if job != nil {
		d := &interfaces.Deployment{
			Name:            *job.Name,
			Namespace:       *job.Namespace,
			Meta:            job.Meta,
			Instances:       *job.TaskGroups[0].Count,
			ResourceVersion: fmt.Sprintf("%d", *job.Version),
		}

		return d, err
	}

	return nil, err
}

// GetDeploymentWithSelector returns the first deployment whos name and namespace match the given
// regular expression and namespace.
func (ni *NomadImpl) GetDeploymentWithSelector(ctx context.Context, selector, namespace string) (*interfaces.Deployment, error) {
	job, err := ni.GetJobWithSelector(ctx, selector, namespace)

	if job != nil {
		d := &interfaces.Deployment{
			Name:            *job.Name,
			Namespace:       *job.Namespace,
			Meta:            job.Meta,
			Instances:       *job.TaskGroups[0].Count,
			ResourceVersion: fmt.Sprintf("%d", *job.Version),
		}

		return d, err
	}

	return nil, err
}

func (ni *NomadImpl) UpdateDeployment(ctx context.Context, deployment *interfaces.Deployment) error {
	job, err := ni.GetJob(ctx, deployment.Name, deployment.Namespace)
	if err != nil {
		return err
	}

	for _, tg := range job.TaskGroups {
		tg.Count = &deployment.Instances
	}

	ver, _ := strconv.ParseUint(deployment.ResourceVersion, 2, 64)

	job.Meta = deployment.Meta
	job.Version = &ver

	return ni.UpsertJob(ctx, job)
}

// CloneDeployment creates a clone of the existing deployment using the details provided in new deployment
func (ni *NomadImpl) CloneDeployment(ctx context.Context, existingDeployment *interfaces.Deployment, newDeployment *interfaces.Deployment) error {
	job, err := ni.GetJob(ctx, existingDeployment.Name, existingDeployment.Namespace)
	if err != nil {
		return err
	}

	for _, tg := range job.TaskGroups {
		tg.Count = &newDeployment.Instances
	}

	job.Meta = newDeployment.Meta
	job.Name = &newDeployment.Name
	job.ID = &newDeployment.Name

	// add the meta to the consul services for the consul-release-controller-version
	// this indicates that the job is managed by the controller and is the primary job
	// we use this in the selector

	if job.Meta[interfaces.RuntimeDeploymentVersionLabel] != "" {
		// add the tag if not already there
		for _, tg := range job.TaskGroups {
			for _, s := range tg.Services {
				hasTag := false
				for _, t := range s.Tags {
					if t == interfaces.RuntimeDeploymentVersionLabel {
						hasTag = true
					}
				}

				if !hasTag {
					s.Tags = append(s.Tags, interfaces.RuntimeDeploymentVersionLabel)
				}
			}
		}

	} else {
		// remove the primary tag if set
		for _, tg := range job.TaskGroups {
			for _, s := range tg.Services {
				tags := []string{}
				for _, t := range s.Tags {
					if t != interfaces.RuntimeDeploymentVersionLabel {
						tags = append(tags, t)
					}
				}

				s.Tags = tags
			}
		}
	}

	for _, tg := range job.TaskGroups {
		for _, s := range tg.Services {
			if s.Meta == nil {
				s.Meta = map[string]string{}
			}

			if job.Meta[interfaces.RuntimeDeploymentVersionLabel] != "" {
				// keys can not contain "-" replace this
				s.Meta[strings.Replace(interfaces.RuntimeDeploymentVersionLabel, "-", "_", -1)] = job.Meta[interfaces.RuntimeDeploymentVersionLabel]
			} else {

				// remove the meta for the version if set
				delete(s.Meta, strings.Replace(interfaces.RuntimeDeploymentVersionLabel, "-", "_", -1))
			}
		}
	}

	return ni.UpsertJob(ctx, job)
}

// DeleteDeployment deletes the given Kubernetes Deployment
func (ni *NomadImpl) DeleteDeployment(ctx context.Context, name, namespace string) error {
	return ni.DeleteJob(ctx, name, namespace)
}

// GetHealthyDeployment blocks until a healthy deployment is found or the process times out
// returns the Deployment an a nil error on success
// returns a nil deployment and a ErrDeploymentNotFound error when the deployment does not exist
// returns a nill deployment and a ErrDeploymentNotHealthy error when the deployment exists but is not in a healthy state
// any other error type signifies an internal error
func (ni *NomadImpl) GetHealthyDeployment(ctx context.Context, name, namespace string) (*interfaces.Deployment, error) {
	job, err := ni.GetHealthyJob(ctx, name, namespace)

	if job != nil {
		d := &interfaces.Deployment{
			Name:            *job.Name,
			Namespace:       *job.Namespace,
			Meta:            job.Meta,
			Instances:       *job.TaskGroups[0].Count,
			ResourceVersion: fmt.Sprintf("%d", *job.Version),
		}

		return d, err
	}

	return nil, err
}

// Returns the Consul resolver subset filter that should be used for this runtime to identify candidate instances
func (ni *NomadImpl) CandidateSubsetFilter() string {
	return fmt.Sprintf(`"%s" not in Service.Tags`, interfaces.RuntimeDeploymentVersionLabel)
}

// Returns the Consul resolver subset filter that should be used for this runtime to identify the primary instances
func (ni *NomadImpl) PrimarySubsetFilter() string {
	return fmt.Sprintf(`"%s" in Service.Tags`, interfaces.RuntimeDeploymentVersionLabel)
}
