package interfaces

import "encoding/json"

// Configurable defines the capabilities for a plugin to be configured
// with a raw json payload
type Configurable interface {

	// Configure the plugin with the given json
	Configure(config json.RawMessage) error
}
