package clients

import (
	"fmt"

	"github.com/hashicorp/consul/api"
)

type Consul interface {
	CreateServiceDefaults(name string) error
	CreateServiceResolver(name string) error
	CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error
	CreateServiceRouter(name string) error
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

func (c *ConsulImpl) CreateServiceDefaults(name string) error {
	defaults := &api.ServiceConfigEntry{}
	defaults.Name = name
	defaults.Kind = api.ServiceDefaults
	defaults.Protocol = "http"

	_, _, err := c.client.ConfigEntries().Set(defaults, &api.WriteOptions{})

	return err
}

func (c *ConsulImpl) CreateServiceResolver(name string) error {
	defaults := &api.ServiceResolverConfigEntry{}
	defaults.Name = name
	defaults.Kind = api.ServiceResolver
	defaults.DefaultSubset = fmt.Sprintf("%s-primary", name)

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

	_, _, err := c.client.ConfigEntries().Set(defaults, &api.WriteOptions{})

	return err
}

func (c *ConsulImpl) CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error {
	defaults := &api.ServiceSplitterConfigEntry{}
	defaults.Kind = api.ServiceSplitter
	defaults.Name = name

	primarySplit := api.ServiceSplit{}
	primarySplit.Service = name
	primarySplit.ServiceSubset = fmt.Sprintf("%s-primary", name)
	primarySplit.Weight = float32(primaryTraffic)

	canarySplit := api.ServiceSplit{}
	canarySplit.Service = name
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

	primaryRoute := api.ServiceRoute{}

	primaryRouteHTTP := &api.ServiceRouteMatch{}
	primaryRouteHTTP.HTTP = &api.ServiceRouteHTTPMatch{
		Header: []api.ServiceRouteHTTPMatchHeader{
			api.ServiceRouteHTTPMatchHeader{Name: "x-primary", Exact: "true"},
		},
	}

	primaryRoute.Destination = &api.ServiceRouteDestination{
		Service:       name,
		ServiceSubset: fmt.Sprintf("%s-primary", name),
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
		Service:       name,
		ServiceSubset: fmt.Sprintf("%s-canary", name),
	}

	canaryRoute.Match = canaryRouteHTTP

	defaultRoute := api.ServiceRoute{}

	defaultRouteHTTP := &api.ServiceRouteMatch{}
	defaultRouteHTTP.HTTP = &api.ServiceRouteHTTPMatch{}

	defaultRoute.Destination = &api.ServiceRouteDestination{
		Service: name,
	}

	defaultRoute.Match = defaultRouteHTTP

	defaults.Routes = []api.ServiceRoute{canaryRoute, primaryRoute, defaultRoute}

	_, _, err := c.client.ConfigEntries().Set(defaults, &api.WriteOptions{})
	return err
}
