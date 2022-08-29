package clients

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
)

const (
	MetaCreatedTag        = "external-source"
	MetaCreatedValue      = "consul-release-controller"
	SubsetPrefix          = "crc"
	UpstreamRouterName    = "consul-release-controller-upstreams"
	ControllerServiceName = "consul-release-controller"
)

type Consul interface {
	// CreateServiceDefaults creates a HTTP protocol service defaults for the given service if it does
	// not already exist. If the defaults already exist the protocol is update to HTTP.
	CreateServiceDefaults(name string) error

	// CreateServiceResolver creates or updates a service resolver for the given service
	// if the ServiceResolver exists then CreateServiceResolver updates it to add the
	// subsets for the canary and primary services
	CreateServiceResolver(name, primarySubsetFilter, candidateSubsetFilter string) error

	// CreateServiceSplitter creates a service splitter for the given name and set the traffic
	// for the primary and the candidate
	CreateServiceSplitter(name string, primaryTraffic, candidateTraffic int) error

	// CreateServiceRoutere creates or updates an existing service router for the given service
	// routes are added to add retries for connection failures
	CreateServiceRouter(name string) error

	// CreateUpstreamRouter creates or updates a service router that allows the candidate services
	// to be called by specifying the correct HOST header
	CreateUpstreamRouter(name string) error

	// CreateServiceIntention creates or updates a service intention that allows the release controller
	// permission to talk to the upstream service. This is only required when a PostDeploymentTest has
	// been configured
	CreateServiceIntention(name string) error

	// DeleteServiceDefaults deletes the service defaults only when they were created by the release controller
	DeleteServiceDefaults(name string) error

	// DeleteServiceResolver removes the service resolver, if the resolver was not created by the release controller
	// this method restores the resolver to the original state
	DeleteServiceResolver(name string) error

	// DeleteServiceSplitter removes the service splitter resource created by the resolver
	DeleteServiceSplitter(name string) error

	// DeleteServiceResolver removes the service resolver, if the resolver was not created by the release controller
	// this method restores the resolver to the original state
	DeleteServiceRouter(name string) error

	// DeleteServiceIntention removes any service intention allowing the release controller communication with the given service
	DeleteServiceIntention(name string) error

	// DeleteUpstreamRouter removes the upstream router that allows the controller to contact candidate services.
	DeleteUpstreamRouter(name string) error

	// Check the Consul health of the service, returns an error when one or more endpoints are not healthy
	// can accept a filter string to return a subset of a services instances https://www.consul.io/api-docs/health#filtering-2
	// Returns an error if all health checks are not passing or if no service instances are found
	CheckHealth(name string, filter string) error

	// SetKV sets the data at the given path in the Consul Key Value store
	SetKV(path string, data []byte) error

	// GetKV gets the data at the given path in the Consul Key Value store
	GetKV(path string) ([]byte, error)

	// DeleteKV deletes the data at the given path in the Consul Key Value store
	DeleteKV(path string) error

	// ListKV returns keynames at the given path
	ListKV(path string) ([]string, error)
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
	wo := &api.WriteOptions{}
	defaults := &api.ServiceConfigEntry{}

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
		wo.Namespace = c.options.Namespace
		defaults.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
		wo.Partition = c.options.Partition
		defaults.Partition = c.options.Partition
	}

	// first check to see if the config already exists,
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceDefaults, name, qo)
	if err != nil {
		// is the item not found if so the error will contain a 404
		if !strings.Contains(err.Error(), "404") {
			return err
		}
	}

	// item exists
	if ce != nil {
		// if the service type is gRPC do not change it and return an error
		switch ce.(*api.ServiceConfigEntry).Protocol {
		case "grpc":
			// release controller should not try to change the protocol of an existing service, return an error
			return fmt.Errorf(
				`service %s has an existing protocol of gRPC, consul release controller can not set protocol to HTTP. 
				please remove the existing Service Defaults before configuring a release`,
				name,
			)
		case "http2":
			// release controller should not try to change the protocol of an existing service, return an error
			return fmt.Errorf(
				`service %s has an existing protocol of HTTP2, consul release controller can not set protocol to HTTP. 
				please remove the existing Service Defaults before configuring a release`,
				name,
			)
		case "http":
			// already HTTP nothing to do
			return nil
		case "tcp":
			// update the existing defaults
			ce.(*api.ServiceConfigEntry).Protocol = "http"
			_, _, err := c.client.ConfigEntries().Set(ce, wo)
			return err
		}
	}

	defaults.Name = name
	defaults.Kind = api.ServiceDefaults
	defaults.Protocol = "http"
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}

	_, _, err = c.client.ConfigEntries().Set(defaults, wo)

	return err
}

func (c *ConsulImpl) CreateServiceResolver(name, primarySubsetFilter, candidateSubsetFilter string) error {
	defaults := &api.ServiceResolverConfigEntry{}
	qo := &api.QueryOptions{}
	wo := &api.WriteOptions{}

	defaults.Name = name
	defaults.Kind = api.ServiceResolver
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}
	defaults.Subsets = map[string]api.ServiceResolverSubset{}

	// this is set to the candidate as until the primary has been created any existing
	// deployments will not have been renamed and will resolve to the candidate selector
	defaults.DefaultSubset = fmt.Sprintf("%s-%s-candidate", SubsetPrefix, name)

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
		wo.Namespace = c.options.Namespace
		defaults.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
		wo.Partition = c.options.Partition
		defaults.Partition = c.options.Partition
	}

	// check that we created this
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceResolver, name, qo)
	if err != nil {
		// is the item not found if so the error will contain a 404
		if !strings.Contains(err.Error(), "404") {
			return err
		}
	}

	if ce != nil {
		// we have an existing entry, mutate rather than overwrite
		defaults = ce.(*api.ServiceResolverConfigEntry)
	}

	primarySubset := api.ServiceResolverSubset{}
	primarySubset.Filter = primarySubsetFilter
	primarySubset.OnlyPassing = true

	canarySubset := api.ServiceResolverSubset{}
	canarySubset.Filter = candidateSubsetFilter
	canarySubset.OnlyPassing = true

	defaults.Subsets[fmt.Sprintf("%s-%s-primary", SubsetPrefix, name)] = primarySubset
	defaults.Subsets[fmt.Sprintf("%s-%s-candidate", SubsetPrefix, name)] = canarySubset

	_, _, err = c.client.ConfigEntries().Set(defaults, wo)

	return err
}

func (c *ConsulImpl) CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error {
	wo := &api.WriteOptions{}
	defaults := &api.ServiceSplitterConfigEntry{}

	defaults.Kind = api.ServiceSplitter
	defaults.Name = name
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}

	if c.options.Namespace != "" {
		defaults.Namespace = c.options.Namespace
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		defaults.Partition = c.options.Partition
		wo.Partition = c.options.Partition
	}

	primarySplit := api.ServiceSplit{}
	primarySplit.ServiceSubset = fmt.Sprintf("%s-%s-primary", SubsetPrefix, name)
	primarySplit.Weight = float32(primaryTraffic)

	canarySplit := api.ServiceSplit{}
	canarySplit.ServiceSubset = fmt.Sprintf("%s-%s-candidate", SubsetPrefix, name)
	canarySplit.Weight = float32(canaryTraffic)

	defaults.Splits = []api.ServiceSplit{primarySplit, canarySplit}

	_, _, err := c.client.ConfigEntries().Set(defaults, wo)

	return err
}

// CreateServiceRouter creates a new service router
func (c *ConsulImpl) CreateServiceRouter(name string) error {
	qo := &api.QueryOptions{}
	wo := &api.WriteOptions{}
	defaults := &api.ServiceRouterConfigEntry{}

	defaults.Name = name
	defaults.Kind = api.ServiceRouter
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}
	defaults.Routes = []api.ServiceRoute{}

	if c.options.Namespace != "" {
		defaults.Namespace = c.options.Namespace
		qo.Namespace = c.options.Namespace
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		defaults.Partition = c.options.Partition
		qo.Partition = c.options.Partition
		wo.Partition = c.options.Partition
	}

	// check that there is not an existing router, if so use it
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceRouter, name, qo)
	if err != nil {
		// is the item not found if so the error will contain a 404
		if !strings.Contains(err.Error(), "404") {
			return err
		}
	}

	if ce != nil {
		// we have an existing entry, mutate rather than overwrite
		defaults = ce.(*api.ServiceRouterConfigEntry)
	}

	// create the routes
	primaryRoute := api.ServiceRoute{}

	primaryRouteHTTP := &api.ServiceRouteMatch{}
	primaryRouteHTTP.HTTP = &api.ServiceRouteHTTPMatch{
		Header: []api.ServiceRouteHTTPMatchHeader{
			api.ServiceRouteHTTPMatchHeader{Name: "x-primary", Exact: "true"},
		},
	}

	primaryRoute.Destination = &api.ServiceRouteDestination{
		Service:               name,
		ServiceSubset:         fmt.Sprintf("%s-%s-primary", SubsetPrefix, name),
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
			api.ServiceRouteHTTPMatchHeader{Name: "x-candidate", Exact: "true"},
		},
	}

	canaryRoute.Destination = &api.ServiceRouteDestination{
		Service:               name,
		ServiceSubset:         fmt.Sprintf("%s-%s-candidate", SubsetPrefix, name),
		NumRetries:            5,
		RetryOnConnectFailure: true,
		RetryOnStatusCodes:    []uint32{503},
	}

	canaryRoute.Match = canaryRouteHTTP
	defaults.Routes = append(defaults.Routes, canaryRoute)

	_, _, err = c.client.ConfigEntries().Set(defaults, wo)
	return err
}

func (c *ConsulImpl) CreateUpstreamRouter(name string) error {
	defaults := &api.ServiceRouterConfigEntry{}
	defaults.Name = UpstreamRouterName
	defaults.Kind = api.ServiceRouter
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}
	defaults.Routes = []api.ServiceRoute{}
	namespace := "default"

	qo := &api.QueryOptions{}

	if c.options.Namespace != "" {
		defaults.Namespace = c.options.Namespace
		qo.Namespace = c.options.Namespace
		namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		defaults.Partition = c.options.Partition
		qo.Partition = c.options.Partition
	}

	// check that there is not an existing router, if so use it
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceRouter, UpstreamRouterName, qo)
	if err != nil {
		// is the item not found if so the error will contain a 404
		if !strings.Contains(err.Error(), "404") {
			return err
		}
	}

	if ce != nil {
		// we have an existing entry, mutate rather than overwrite
		defaults = ce.(*api.ServiceRouterConfigEntry)
	}

	// create the new route
	candidateRoute := api.ServiceRoute{}

	candidateRouteHTTP := &api.ServiceRouteMatch{}
	candidateRouteHTTP.HTTP = &api.ServiceRouteHTTPMatch{
		Header: []api.ServiceRouteHTTPMatchHeader{
			api.ServiceRouteHTTPMatchHeader{Name: "HOST", Exact: fmt.Sprintf("%s.%s", name, namespace)},
		},
	}

	candidateRoute.Destination = &api.ServiceRouteDestination{
		Service:               name,
		ServiceSubset:         fmt.Sprintf("%s-%s-candidate", SubsetPrefix, name),
		NumRetries:            5,
		RetryOnConnectFailure: true,
		RetryOnStatusCodes:    []uint32{503},
	}

	candidateRoute.Match = candidateRouteHTTP
	defaults.Routes = append(defaults.Routes, candidateRoute)

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

func (c *ConsulImpl) CreateServiceIntention(name string) error {
	defaults := &api.ServiceIntentionsConfigEntry{}
	defaults.Name = name
	defaults.Kind = api.ServiceIntentions
	defaults.Meta = map[string]string{MetaCreatedTag: MetaCreatedValue}
	defaults.Sources = []*api.SourceIntention{}

	// create the intention allowing access for the controller
	i := &api.SourceIntention{}
	i.Name = ControllerServiceName
	i.Action = "allow"

	qo := &api.QueryOptions{}
	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		defaults.Namespace = c.options.Namespace
		qo.Namespace = c.options.Namespace
		wo.Namespace = c.options.Namespace

		// TODO
		// this assumes that the consul release controller is running in the same
		// namespace as the destination service, this needs fixed
		i.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		defaults.Partition = c.options.Partition
		qo.Partition = c.options.Partition
		wo.Partition = c.options.Partition

		// TODO
		// this assumes that the consul release controller is running in the same
		// partition as the destination service, this needs fixed
		i.Partition = c.options.Partition
	}

	// check that there is not an existing intention, if so use it
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceIntentions, name, qo)
	if err != nil {
		// is the item not found if so the error will contain a 404
		if !strings.Contains(err.Error(), "404") {
			return err
		}
	}

	if ce != nil {
		// we have an existing entry, mutate rather than overwrite
		defaults = ce.(*api.ServiceIntentionsConfigEntry)

		// first check to see if the source already exists, if so exit
		for _, s := range defaults.Sources {
			if s.Name == ControllerServiceName {
				// intention already exists, exit
				return nil
			}
		}
	}

	// update the list of intentions adding the controller intention
	defaults.Sources = append(defaults.Sources, i)

	_, _, err = c.client.ConfigEntries().Set(defaults, wo)

	return err
}

func (c *ConsulImpl) DeleteServiceDefaults(name string) error {
	qo := &api.QueryOptions{}
	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
		wo.Partition = c.options.Partition
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

	_, err = c.client.ConfigEntries().Delete("service-defaults", name, wo)
	return err
}

func (c *ConsulImpl) DeleteServiceResolver(name string) error {
	qo := &api.QueryOptions{}
	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
		wo.Partition = c.options.Partition
	}

	// check that we created this
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceResolver, name, qo)
	// no config entry found
	if err != nil && ce != nil {
		return nil
	}

	// internal error
	if err != nil || ce == nil {
		return err
	}

	// if we did not create the resolver do an update removing the subsets
	if ce.GetMeta()[MetaCreatedTag] != MetaCreatedValue {
		delete(ce.(*api.ServiceResolverConfigEntry).Subsets, fmt.Sprintf("%s-%s-primary", SubsetPrefix, name))
		delete(ce.(*api.ServiceResolverConfigEntry).Subsets, fmt.Sprintf("%s-%s-candidate", SubsetPrefix, name))

		_, _, err := c.client.ConfigEntries().Set(ce, wo)
		return err
	}

	_, err = c.client.ConfigEntries().Delete(api.ServiceResolver, name, wo)
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
	qo := &api.QueryOptions{}
	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
		wo.Partition = c.options.Partition
	}

	// check that we created this
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceRouter, name, qo)
	if err != nil && ce != nil {
		return nil
	}

	if err != nil || ce == nil {
		return err
	}

	// if we did not create the router do an update removing the routes for the primary and candidate
	if ce.GetMeta()[MetaCreatedTag] != MetaCreatedValue {
		routes := []api.ServiceRoute{}

		for _, r := range ce.(*api.ServiceRouterConfigEntry).Routes {
			if r.Destination.ServiceSubset != fmt.Sprintf("%s-%s-primary", SubsetPrefix, name) &&
				r.Destination.ServiceSubset != fmt.Sprintf("%s-%s-candidate", SubsetPrefix, name) {
				routes = append(routes, r)
			}
		}

		_, _, err := c.client.ConfigEntries().Set(ce, wo)
		return err
	}

	_, err = c.client.ConfigEntries().Delete(api.ServiceRouter, name, wo)
	return err
}

func (c *ConsulImpl) DeleteUpstreamRouter(name string) error {
	qo := &api.QueryOptions{}
	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
		wo.Partition = c.options.Partition
	}

	ce, _, err := c.client.ConfigEntries().Get(api.ServiceRouter, UpstreamRouterName, qo)
	if err != nil && ce != nil {
		return nil
	}

	if err != nil || ce == nil {
		return err
	}

	sre := ce.(*api.ServiceRouterConfigEntry)

	// remove the route for this service
	routes := []api.ServiceRoute{}

	for _, r := range sre.Routes {
		if r.Destination.Service != name {
			routes = append(routes, r)
		}
	}

	// no routes left, clean up config
	if len(routes) == 0 {
		_, err = c.client.ConfigEntries().Delete(api.ServiceRouter, UpstreamRouterName, wo)
		return err
	}

	// update the config
	sre.Routes = routes
	_, _, err = c.client.ConfigEntries().Set(sre, wo)
	return err
}

func (c *ConsulImpl) DeleteServiceIntention(name string) error {
	qo := &api.QueryOptions{}
	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
		wo.Partition = c.options.Partition
	}

	// check that there is not an existing intention, if so use it
	ce, _, err := c.client.ConfigEntries().Get(api.ServiceIntentions, name, qo)
	if err != nil && ce != nil {
		return nil
	}

	if err != nil || ce == nil {
		return err
	}

	// if we did not create the intention do an update removing the allow for the release controller
	if ce.GetMeta()[MetaCreatedTag] != MetaCreatedValue {
		sources := []*api.SourceIntention{}
		for _, s := range ce.(*api.ServiceIntentionsConfigEntry).Sources {
			if s.Name != ControllerServiceName {
				sources = append(sources, s)
			}
		}

		// update the intention
		ce.(*api.ServiceIntentionsConfigEntry).Sources = sources
		_, _, err = c.client.ConfigEntries().Set(ce, wo)
		return err
	}

	// delete the intention
	_, err = c.client.ConfigEntries().Delete(api.ServiceIntentions, name, wo)
	return err
}

// CheckHealth returns an error if the named service has any health checks that are failing
func (c *ConsulImpl) CheckHealth(name string, filter string) error {
	qo := &api.QueryOptions{Filter: filter}

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

	if len(checks) == 0 {
		return fmt.Errorf("no service checks returned for service %s, with filter %s", name, filter)
	}

	// check the connect health
	for _, chk := range checks {
		if chk.Checks.AggregatedStatus() != "passing" {
			return fmt.Errorf("service health checks failing: %s, %s", chk.Service.ID, chk.Checks.AggregatedStatus())
		}
	}

	// Also check the connect health to ensure that the proxy is healthy
	checks, _, err = c.client.Health().Connect(name, "", false, qo)
	if err != nil {
		return fmt.Errorf("unable to check health for service %s: %s", name, err)
	}

	if len(checks) == 0 {
		return fmt.Errorf("no service checks returned for service %s, with filter %s", name, filter)
	}

	for _, chk := range checks {
		if chk.Checks.AggregatedStatus() != "passing" {
			return fmt.Errorf("connect health checks failing: %s, %s", chk.Service.ID, chk.Checks.AggregatedStatus())
		}
	}

	return nil
}

// SetKV sets the data at the given path in the Consul Key Value store
func (c *ConsulImpl) SetKV(path string, data []byte) error {
	kvp := &api.KVPair{}
	if c.options.Namespace != "" {
		kvp.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		kvp.Partition = c.options.Partition
	}

	kvp.Key = path
	kvp.Value = data

	_, err := c.client.KV().Put(kvp, &api.WriteOptions{})
	if err != nil {
		return fmt.Errorf("unable to write data to Consul KV: %s", err)
	}

	return nil
}

// GetKV gets the data at the given path in the Consul Key Value store
// When the key at the path does not exist, this function returns a nil data payload
// and no error.
func (c *ConsulImpl) GetKV(path string) ([]byte, error) {
	qo := &api.QueryOptions{}

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
	}

	kp, _, err := c.client.KV().Get(path, qo)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch key from path %s in Consul KV: %s", path, err)
	}

	// no key found
	if kp == nil {
		return nil, nil
	}

	return kp.Value, nil
}

// DeleteKV deletes the data at the given path in the Consul Key Value store
func (c *ConsulImpl) DeleteKV(path string) error {
	wo := &api.WriteOptions{}

	if c.options.Namespace != "" {
		wo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		wo.Partition = c.options.Partition
	}

	_, err := c.client.KV().DeleteTree(path, wo)
	if err != nil {
		return fmt.Errorf("unable to delete key from path %s in Consul KV: %s", path, err)
	}

	return nil
}

// ListKV returns keynames at the given path
func (c *ConsulImpl) ListKV(path string) ([]string, error) {
	qo := &api.QueryOptions{}

	if c.options.Namespace != "" {
		qo.Namespace = c.options.Namespace
	}

	if c.options.Partition != "" {
		qo.Partition = c.options.Partition
	}

	k, _, err := c.client.KV().Keys(path, "", qo)
	if err != nil {
		return nil, err
	}

	return k, nil
}
