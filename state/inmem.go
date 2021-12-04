package state

import (
	"errors"
	"sync"

	"github.com/nicholasjackson/consul-canary-controller/models"
)

var ReleaseNotFound = errors.New("Release not found")

type InmemStore struct {
	m        sync.Mutex
	releases []*models.Release
}

func NewInmemStore() *InmemStore {
	return &InmemStore{m: sync.Mutex{}, releases: []*models.Release{}}
}

func (m *InmemStore) UpsertRelease(d *models.Release) error {
	m.m.Lock()
	defer m.m.Unlock()

	m.releases = append(m.releases, d)

	return nil
}

func (m *InmemStore) ListReleases(options *ListOptions) ([]*models.Release, error) {
	m.m.Lock()
	defer m.m.Unlock()

	if options == nil {
		return m.releases, nil
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

	index := -1
	// find the correct deployment
	for i, d := range m.releases {
		if d.Name == name {
			index = i
			break
		}
	}

	m.releases = append(m.releases[:index], m.releases[index+1:]...)

	if index == -1 {
		return ReleaseNotFound
	}

	return nil
}
