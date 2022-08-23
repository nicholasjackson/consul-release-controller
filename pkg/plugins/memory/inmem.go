package memory

import (
	"fmt"
	"sync"

	"github.com/nicholasjackson/consul-release-controller/pkg/models"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
)

type Store struct {
	m        sync.Mutex
	releases map[string]*models.Release
}

func NewStore() *Store {
	return &Store{m: sync.Mutex{}, releases: map[string]*models.Release{}}
}

func (m *Store) UpsertRelease(d *models.Release) error {
	if d.Name == "" {
		return fmt.Errorf("invalid name: %s for release", d.Name)
	}

	m.m.Lock()
	defer m.m.Unlock()

	m.releases[d.Name] = d

	return nil
}

func (m *Store) ListReleases(options *interfaces.ListOptions) ([]*models.Release, error) {
	m.m.Lock()
	defer m.m.Unlock()

	if options == nil {
		ret := []*models.Release{}
		for _, r := range m.releases {
			ret = append(ret, r)
		}

		return ret, nil
	}

	// filter the releases based on options
	ret := []*models.Release{}
	for _, r := range m.releases {
		if r.Runtime.Name == options.Runtime {
			ret = append(ret, r)
		}
	}

	return ret, nil
}

func (m *Store) DeleteRelease(name string) error {
	m.m.Lock()
	defer m.m.Unlock()

	r, ok := m.releases[name]
	if !ok {
		return interfaces.ReleaseNotFound
	}

	delete(m.releases, r.Name)

	return nil
}

func (m *Store) GetRelease(name string) (*models.Release, error) {
	m.m.Lock()
	defer m.m.Unlock()

	for _, r := range m.releases {
		if r.Name == name {
			return r, nil
		}
	}

	return nil, interfaces.ReleaseNotFound
}
