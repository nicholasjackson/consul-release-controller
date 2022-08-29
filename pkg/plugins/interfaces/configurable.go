package interfaces

import (
	"encoding/json"

	"github.com/hashicorp/go-hclog"
)

// Configurable defines the capabilities for a plugin to be configured
// with a raw json payload
type Configurable interface {

	// Configure the plugin with the given json config from the Release config.
	// The plugin creator also injects a logger that can be used by plugin authors to log
	// to the main controller log file and also a store object that can be used to
	// write and retrieve the plugins state from the configured datastore.
	Configure(config json.RawMessage, log hclog.Logger, store PluginStateStore) error
}
