package clients

type Consul interface {
	CreateServiceDefaults(name string) error
	CreateServiceResolver(name string) error
	CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error
	CreateServiceRouter(name string) error
}

func NewConsul() (Consul, error) {
	return &ConsulImpl{}, nil
}

type ConsulImpl struct{}

func (c *ConsulImpl) CreateServiceDefaults(name string) error {
	return nil
}

func (c *ConsulImpl) CreateServiceResolver(name string) error {
	return nil
}

func (c *ConsulImpl) CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error {
	return nil
}

func (c *ConsulImpl) CreateServiceRouter(name string) error {
	return nil
}
