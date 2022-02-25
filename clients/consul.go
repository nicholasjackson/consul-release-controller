package clients

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
)

const (
	MetaCreatedTag   = "created-by"
	MetaCreatedValue = "consul-release-controller"
)

type Consul interface {
	CreateServiceDefaults(name string) error
	CreateServiceResolver(name string) error
	CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error
	CreateServiceRouter(name string, onlyDefault bool) error

	DeleteServiceDefaults(name string) error
	DeleteServiceResolver(name string) error
	DeleteServiceSplitter(name string) error
	DeleteServiceRouter(name string) error

	CheckHealth(name string, t interfaces.ServiceVariant) error
}

type ConsulOptions struct {
	Namespace string // Enterprise only
	Partition string // Enterprise only
}

func NewConsul(options *ConsulOptions) (Consul, error) {
	// Get a new client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	if options == nil {
		options = &ConsulOptions{}
	}

	return &ConsulImpl{client, options}, nil
}

type ConsulImpl struct {
	client  *api.Client
	options *ConsulOptions
}

// CreateServiceDefaults if does not exist
func (c *ConsulImpl) CreateServiceDefaults(name string) error {
	qo := &api.QueryOptions{}

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
	}

	// first check to see if the config already exists,
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceDefaults, name, qo)
	if err != nil {
		// is the item not found if so the error will contain a 404
		if !strings.Contains(err.Error(), "404") {
			return err
		}
	}

	// item exists do not create
	if ce != nil {
		return nil
	}

	defaults := &api.ServiceConfigEntry{}
	defaults.Name = name
	defaults.Kind = api.ServiceDefaults
	defaults.Protocol = "http"
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}

	if c.options.Namespace != "" {
		defaults.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		defaults.Partition = c.options.Partition
	}

	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		wo.Partition = c.options.Partition
	}

	_, _, err = c.client.ConfigEntries().Set(defaults, wo)

	return err
}

func (c *ConsulImpl) CreateServiceResolver(name string) error {
	defaults := &api.ServiceResolverConfigEntry{}
	defaults.Name = name
	defaults.Kind = api.ServiceResolver
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}
	defaults.DefaultSubset = fmt.Sprintf("%s-canary", name)

	if c.options.Namespace != "" {
		defaults.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		defaults.Partition = c.options.Partition
	}

	primarySubset := &api.ServiceResolverSubset{}
	primarySubset.Filter = fmt.Sprintf(`Service.ID contains "%s-deployment-primary"`, name)
	primarySubset.OnlyPassing = true

	canarySubset := &api.ServiceResolverSubset{}
	canarySubset.Filter = fmt.Sprintf(`Service.ID not contains "%s-deployment-primary"`, name)
	canarySubset.OnlyPassing = true

	defaults.Subsets = map[string]api.ServiceResolverSubset{
		fmt.Sprintf("%s-primary", name): *primarySubset,
		fmt.Sprintf("%s-canary", name):  *canarySubset,
	}

	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		wo.Partition = c.options.Partition
	}

	_, _, err := c.client.ConfigEntries().Set(defaults, wo)

	return err
}

func (c *ConsulImpl) CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error {
	defaults := &api.ServiceSplitterConfigEntry{}
	defaults.Kind = api.ServiceSplitter
	defaults.Name = name
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}

	if c.options.Namespace != "" {
		defaults.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		defaults.Partition = c.options.Partition
	}

	primarySplit := api.ServiceSplit{}
	primarySplit.ServiceSubset = fmt.Sprintf("%s-primary", name)
	primarySplit.Weight = float32(primaryTraffic)

	canarySplit := api.ServiceSplit{}
	canarySplit.ServiceSubset = fmt.Sprintf("%s-canary", name)
	canarySplit.Weight = float32(canaryTraffic)

	defaults.Splits = []api.ServiceSplit{primarySplit, canarySplit}

	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		wo.Partition = c.options.Partition
	}

	_, _, err := c.client.ConfigEntries().Set(defaults, wo)

	return err
}

// CreateServiceRouter creates a new service router, if the onlyDefault option is specified
// only the default route is created.
func (c *ConsulImpl) CreateServiceRouter(name string, onlyDefault bool) error {
	defaults := &api.ServiceRouterConfigEntry{}
	defaults.Name = name
	defaults.Kind = api.ServiceRouter
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}
	defaults.Routes = []api.ServiceRoute{}

	if c.options.Namespace != "" {
		defaults.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		defaults.Partition = c.options.Partition
	}

	if !onlyDefault {
		primaryRoute := api.ServiceRoute{}

		primaryRouteHTTP := &api.ServiceRouteMatch{}
		primaryRouteHTTP.HTTP = &api.ServiceRouteHTTPMatch{
			Header: []api.ServiceRouteHTTPMatchHeader{
				api.ServiceRouteHTTPMatchHeader{Name: "x-primary", Exact: "true"},
			},
		}

		primaryRoute.Destination = &api.ServiceRouteDestination{
			Service:               name,
			ServiceSubset:         fmt.Sprintf("%s-primary", name),
			NumRetries:            5,
			RetryOnConnectFailure: true,
			RetryOnStatusCodes:    []uint32{503},
		}

		primaryRoute.Match = primaryRouteHTTP
		defaults.Routes = append(defaults.Routes, primaryRoute)

		canaryRoute := api.ServiceRoute{}

		canaryRouteHTTP := &api.ServiceRouteMatch{}
		canaryRouteHTTP.HTTP = &api.ServiceRouteHTTPMatch{
			Header: []api.ServiceRouteHTTPMatchHeader{
				api.ServiceRouteHTTPMatchHeader{Name: "x-canary", Exact: "true"},
			},
		}

		canaryRoute.Destination = &api.ServiceRouteDestination{
			Service:               name,
			ServiceSubset:         fmt.Sprintf("%s-canary", name),
			NumRetries:            5,
			RetryOnConnectFailure: true,
			RetryOnStatusCodes:    []uint32{503},
		}

		canaryRoute.Match = canaryRouteHTTP
		defaults.Routes = append(defaults.Routes, canaryRoute)
	}

	defaultRoute := api.ServiceRoute{}

	defaultRouteHTTP := &api.ServiceRouteMatch{}
	defaultRouteHTTP.HTTP = &api.ServiceRouteHTTPMatch{}

	defaultRoute.Destination = &api.ServiceRouteDestination{
		Service:               name,
		NumRetries:            5,
		RetryOnConnectFailure: true,
		RetryOnStatusCodes:    []uint32{503},
	}

	defaultRoute.Match = defaultRouteHTTP

	defaults.Routes = append(defaults.Routes, defaultRoute)

	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		wo.Partition = c.options.Partition
	}
	_, _, err := c.client.ConfigEntries().Set(defaults, wo)
	return err
}

func (c *ConsulImpl) DeleteServiceDefaults(name string) error {
	qo := &api.QueryOptions{}

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
	}

	// check that we created this
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceDefaults, name, qo)
	if err != nil && ce != nil {
		return nil
	}

	if err != nil || ce == nil {
		return err
	}

	// if we did not create this return
	if ce.GetMeta()[MetaCreatedTag] != MetaCreatedValue {
		return nil
	}

	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		wo.Partition = c.options.Partition
	}

	_, err = c.client.ConfigEntries().Delete("service-defaults", name, wo)
	return err
}

func (c *ConsulImpl) DeleteServiceResolver(name string) error {
	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		wo.Partition = c.options.Partition
	}

	_, err := c.client.ConfigEntries().Delete("service-resolver", name, wo)
	return err
}

func (c *ConsulImpl) DeleteServiceSplitter(name string) error {
	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		wo.Partition = c.options.Partition
	}

	_, err := c.client.ConfigEntries().Delete("service-splitter", name, wo)
	return err
}

func (c *ConsulImpl) DeleteServiceRouter(name string) error {
	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		wo.Partition = c.options.Partition
	}

	_, err := c.client.ConfigEntries().Delete("service-router", name, wo)
	return err
}

// CheckHealth returns an error if the named service has any health checks that are failing
func (c *ConsulImpl) CheckHealth(name string, t interfaces.ServiceVariant) error {
	qo := &api.QueryOptions{}

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
	}

	checks, _, err := c.client.Health().Service(name, "", false, qo)
	if err != nil {
		return fmt.Errorf("unable to check health for service %s: %s", name, err)
	}

	for _, chk := range checks {
		if t == interfaces.Primary && !strings.Contains("primary", chk.Service.ID) {
			break
		}

		if t == interfaces.Candidate && strings.Contains("primary", chk.Service.ID) {
			break
		}

		if chk.Checks.AggregatedStatus() != "passing" {
			return fmt.Errorf("service health checks failing: %s, %s", chk.Service.ID, chk.Checks.AggregatedStatus())
		}
	}

	return nil
}
