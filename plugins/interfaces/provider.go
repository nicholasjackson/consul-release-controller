package interfaces

import "github.com/hashicorp/go-hclog"

// Provider loads and creates registered plugins
type Provider interface {
	// CreateReleaser returns a Setup plugin that corresponds sto the given name
	CreateReleaser(pluginName string) (Releaser, error)

	// CreateRuntime returns a Runtime plugin that corresponds to the given name
	CreateRuntime(pluginName string) (Runtime, error)

	// CreateMonitoring returns a Monitor plugin that corresponds to the given name
	CreateMonitor(pluginName string) (Monitor, error)

	// CreateStrategy returns a Strategy plugin that corresponds to the given name
	// Strategy is responsible for checking metrics to determine health, it requires a
	// Monitor plugin in order to do this
	CreateStrategy(pluginName string, mp Monitor) (Strategy, error)

	// Gets an instance of the current logger
	GetLogger() hclog.Logger
}
