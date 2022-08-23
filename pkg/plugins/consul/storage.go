package consul

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DisgoOrg/log"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/clients"
	"github.com/nicholasjackson/consul-release-controller/pkg/models"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
)

type Storage struct {
	log          hclog.Logger
	consulClient clients.Consul

	// used by pluginStateStore
	releaseName string
	pluginName  string
}

const basePath = "consul-release-controller/releases"
const configPath = "config"
const pluginPath = "plugin-state"

func NewStorage(l hclog.Logger) (*Storage, error) {
	opts := &clients.ConsulOptions{}

	// create a new Consul client
	cc, err := clients.NewConsul(opts)
	if err != nil {
		return nil, err
	}

	return &Storage{log: l, consulClient: cc}, nil
}

func (s *Storage) CreatePluginStateStore(r *models.Release, pluginName string) interfaces.PluginStateStore {
	return &Storage{s.log, s.consulClient, r.Name, pluginName}
}

// UpsertRelease creates a new release if not already existing, or updates and existing release
func (s *Storage) UpsertRelease(r *models.Release) error {
	d, err := json.MarshalIndent(r, "", " ")
	if err != nil {
		return err
	}

	log.Debug("Upserting release in consul", "name", r.Name)

	return s.consulClient.SetKV(releaseConfigPath(r.Name), d)
}

// ListReleases returns the releases in the data store that match the given options
// if options is nil then all releases are returned
func (s *Storage) ListReleases(options *interfaces.ListOptions) ([]*models.Release, error) {
	keys, err := s.consulClient.ListKV(basePath)
	if err != nil {
		return nil, err
	}
	s.log.Debug("Fetched keys from Consul", "keys", keys)

	releases := []*models.Release{}
	for _, k := range keys {
		// Consul will return all they keys not the paths, we only need to check
		// keys that end in the path configPath
		if strings.HasSuffix(k, configPath) {

			s.log.Debug("Fetching release from Consul", "path", k)

			rel, err := s.getRelease(k)
			if err != nil {
				return nil, err
			}

			// filter the response
			if options == nil || (rel.Runtime != nil && options.Runtime == rel.Runtime.Name) {
				releases = append(releases, rel)
			}
		}
	}

	return releases, nil
}

// GetRelease with the given name
// Returns a nil Release and ReleaseNotFound error when a Release with the given name does not
// exist in the store.
// Any other error indicates an internal problem fetching the Release
func (s *Storage) GetRelease(name string) (*models.Release, error) {
	return s.getRelease(releaseConfigPath(name))
}

func (s *Storage) getRelease(path string) (*models.Release, error) {
	d, err := s.consulClient.GetKV(path)
	if err != nil {
		return nil, err
	}

	if len(d) == 0 {
		return nil, interfaces.ReleaseNotFound
	}

	rel := &models.Release{}
	err = json.Unmarshal(d, rel)
	if err != nil {
		return nil, err
	}

	return rel, nil
}

// DeleteRelease with the given name
func (s *Storage) DeleteRelease(name string) error {
	path := releasePath(name)

	s.log.Debug("Deleting release from Consul", "path", path)
	return s.consulClient.DeleteKV(path)
}

func (s *Storage) UpsertState(data []byte) error {
	if s.pluginName == "" || s.releaseName == "" {
		return fmt.Errorf("storage incorrectly configured, no pluginName or releaseName")
	}

	// get the path for the state
	sp := stateConfigPath(s.releaseName, s.pluginName)

	return s.consulClient.SetKV(sp, data)
}

func (s *Storage) GetState() ([]byte, error) {
	if s.pluginName == "" || s.releaseName == "" {
		return nil, fmt.Errorf("storage incorrectly configured, no pluginName or releaseName")
	}

	// get the path for the state
	sp := stateConfigPath(s.releaseName, s.pluginName)

	d, err := s.consulClient.GetKV(sp)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve key from Consul: %s", err)
	}

	// if the key is not found the consul client returns a nil payload
	if len(d) == 0 {
		return nil, interfaces.PluginStateNotFound
	}

	return d, nil
}

// stateConfigPath is a helper that
func stateConfigPath(name, pluginName string) string {
	return fmt.Sprintf("%s/%s/%s/%s", basePath, name, pluginPath, pluginName)
}

// releaseConfigPath is a helper that
func releaseConfigPath(name string) string {
	return fmt.Sprintf("%s/%s/%s", basePath, name, configPath)
}

func releasePath(name string) string {
	return fmt.Sprintf("%s/%s", basePath, name)
}
