package state

import (
	"errors"
	"fmt"
	"sync"

	"github.com/nicholasjackson/consul-release-controller/models"
)

var ReleaseNotFound = errors.New("Release not found")

type InmemStore struct {
	m        sync.Mutex
	releases map[string]*models.Release
}

func NewInmemStore() *InmemStore {
	return &InmemStore{m: sync.Mutex{}, releases: map[string]*models.Release{}}
}

func (m *InmemStore) UpsertRelease(d *models.Release) error {
	if d.Name == "" {
		return fmt.Errorf("invalid name: %s for release", d.Name)
	}

	m.m.Lock()
	defer m.m.Unlock()

	m.releases[d.Name] = d

	return nil
}

func (m *InmemStore) ListReleases(options *ListOptions) ([]*models.Release, error) {
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

func (m *InmemStore) DeleteRelease(name string) error {
	m.m.Lock()
	defer m.m.Unlock()

	r, ok := m.releases[name]
	if !ok {
		return ReleaseNotFound
	}

	delete(m.releases, r.Name)

	return nil
}

func (m *InmemStore) GetRelease(name string) (*models.Release, error) {
	m.m.Lock()
	defer m.m.Unlock()

	for _, r := range m.releases {
		if r.Name == name {
			return r, nil
		}
	}

	return nil, ReleaseNotFound
}
