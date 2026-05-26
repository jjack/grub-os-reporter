package servicemanager

import (
	"sort"

	"github.com/rs/zerolog/log"
)

type Factory func() Manager

type Registry struct {
	services map[string]Factory
}

func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]Factory),
	}
}

func (r *Registry) Register(name string, factory Factory) {
	r.services[name] = factory
}

func (r *Registry) Get(name string) Manager {
	if factory, ok := r.services[name]; ok {
		return factory()
	}
	return nil
}

func (r *Registry) Detect() (Manager, error) {
	var names []string
	for name := range r.services {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		mgr := r.services[name]()
		if mgr.IsActive() {
			log.Debug().Str("name", name).Msg("Detected service manager")
			return mgr, nil
		}
	}
	return nil, ErrNotSupported
}

func (r *Registry) ActiveServices() []string {
	var names []string
	for name, factory := range r.services {
		if factory().IsActive() {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (r *Registry) SupportedServices() []string {
	var names []string
	for name := range r.services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
