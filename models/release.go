package models

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Release struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`

	Version string `json:"version"`

	Created     time.Time `json:"created"`
	LastUpdated time.Time `json:"last_updated"`

	Releaser *PluginConfig `json:"releaser"`
	Runtime  *PluginConfig `json:"runtime"`
	Strategy *PluginConfig `json:"strategy"`
	Monitor  *PluginConfig `json:"monitor"`
}

type PluginConfig struct {
	Name   string          `json:"plugin_name"`
	Config json.RawMessage `json:"config"`
}

// FromJsonBody decodes the json body into the Deployment type
func (d *Release) FromJsonBody(r io.ReadCloser) error {
	if r == nil {
		return fmt.Errorf("no json body provided")
	}

	return json.NewDecoder(r).Decode(d)
}

// ToJson serializes the deployment to json
func (d *Release) ToJson() []byte {
	data, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}

	return data
}
