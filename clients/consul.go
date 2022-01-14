package clients

import (
	"fmt"

	"github.com/hashicorp/consul/api"
)

const (
	MetaCreatedTag   = "created-by"
	MetaCreatedValue = "consul-release-controller"
)

type Consul interface {
	CreateServiceDefaults(name string) error
	CreateServiceResolver(name string) error
	CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error
	CreateServiceRouter(name string) error

	DeleteDefaults(name string) error
	DeleteResolver(name string) error
	DeleteSplitter(name string) error
	DeleteRouter(name string) error
}

func NewConsul() (Consul, error) {
	// Get a new client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	return &ConsulImpl{client}, nil
}

type ConsulImpl struct {
	client *api.Client
}

// CreateServiceDefaults if does not exist
func (c *ConsulImpl) CreateServiceDefaults(name string) error {
	// first check to see if the config already exists,
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceDefaults, name, &api.QueryOptions{})
	if err != nil && ce != nil {
		return nil
	}

	defaults := &api.ServiceConfigEntry{}
	defaults.Name = name
	defaults.Kind = api.ServiceDefaults
	defaults.Protocol = "http"
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}

	_, _, err = c.client.ConfigEntries().Set(defaults, &api.WriteOptions{})

	return err
}

func (c *ConsulImpl) CreateServiceResolver(name string) error {
	defaults := &api.ServiceResolverConfigEntry{}
	defaults.Name = name
	defaults.Kind = api.ServiceResolver
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}
	defaults.DefaultSubset = fmt.Sprintf("%s-canary", name)

	primarySubset := &api.ServiceResolverSubset{}
	primarySubset.Filter = fmt.Sprintf(`Service.ID contains "%s-deployment-primary"`, name)
	primarySubset.OnlyPassing = false

	canarySubset := &api.ServiceResolverSubset{}
	canarySubset.Filter = fmt.Sprintf(`Service.ID not contains "%s-deployment-primary"`, name)
	canarySubset.OnlyPassing = false

	defaults.Subsets = map[string]api.ServiceResolverSubset{
		fmt.Sprintf("%s-primary", name): *primarySubset,
		fmt.Sprintf("%s-canary", name):  *canarySubset,
	}

	_, _, err := c.client.ConfigEntries().Set(defaults, &api.WriteOptions{})

	return err
}

func (c *ConsulImpl) CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error {
	defaults := &api.ServiceSplitterConfigEntry{}
	defaults.Kind = api.ServiceSplitter
	defaults.Name = name
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}

	primarySplit := api.ServiceSplit{}
	primarySplit.ServiceSubset = fmt.Sprintf("%s-primary", name)
	primarySplit.Weight = float32(primaryTraffic)

	canarySplit := api.ServiceSplit{}
	canarySplit.ServiceSubset = fmt.Sprintf("%s-canary", name)
	canarySplit.Weight = float32(canaryTraffic)

	defaults.Splits = []api.ServiceSplit{primarySplit, canarySplit}

	_, _, err := c.client.ConfigEntries().Set(defaults, &api.WriteOptions{})

	return err
}

func (c *ConsulImpl) CreateServiceRouter(name string) error {
	defaults := &api.ServiceRouterConfigEntry{}
	defaults.Name = name
	defaults.Kind = api.ServiceRouter
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}

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

	defaults.Routes = []api.ServiceRoute{canaryRoute, primaryRoute, defaultRoute}

	_, _, err := c.client.ConfigEntries().Set(defaults, &api.WriteOptions{})
	return err
}

func (c *ConsulImpl) DeleteDefaults(name string) error {
	// check that we created this
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceDefaults, name, &api.QueryOptions{})
	if err != nil && ce != nil {
		return nil
	}

	if err != nil || ce == nil {
		return err
	}

	// if we did not create this return
	if ce.GetMeta()["created-by"] != "consul-release-controller" {
		return nil
	}

	_, err = c.client.ConfigEntries().Delete("service-defaults", name, &api.WriteOptions{})
	return err
}

func (c *ConsulImpl) DeleteResolver(name string) error {
	_, err := c.client.ConfigEntries().Delete("service-resolver", name, &api.WriteOptions{})
	return err
}

func (c *ConsulImpl) DeleteSplitter(name string) error {
	_, err := c.client.ConfigEntries().Delete("service-splitter", name, &api.WriteOptions{})
	return err
}

func (c *ConsulImpl) DeleteRouter(name string) error {
	_, err := c.client.ConfigEntries().Delete("service-router", name, &api.WriteOptions{})
	return err
}
