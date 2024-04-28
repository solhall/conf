package conf

import (
	"fmt"
	"reflect"
	"slices"
)

var _ Provider = (*PriorityProvider)(nil)

type PriorityProvider struct {
	m         map[string]typ
	providers []Provider
	required  []string
	missing   []string
}

func NewPriorityProvider(providers ...Provider) *PriorityProvider {
	return &PriorityProvider{
		m:         make(map[string]typ),
		providers: providers,
		missing:   []string{},
	}
}

func (p *PriorityProvider) StringVar(to *string, name, fallback string, required bool) {
	for _, provider := range p.providers {
		provider.StringVar(to, name, fallback, required)
	}

	p.m[name] = typ{kind: reflect.String, stringVal: to}
	if required {
		p.required = append(p.required, name)
	}
}

func (p *PriorityProvider) IntVar(to *int, name string, fallback int, required bool) {
	for _, provider := range p.providers {
		provider.IntVar(to, name, fallback, required)
	}

	p.m[name] = typ{kind: reflect.Int, intVal: to}
	if required {
		p.required = append(p.required, name)
	}
}

func (p *PriorityProvider) BoolVar(to *bool, name string, fallback bool, required bool) {
	for _, provider := range p.providers {
		provider.BoolVar(to, name, fallback, required)
	}

	p.m[name] = typ{kind: reflect.Bool, boolVal: to}
	if required {
		p.required = append(p.required, name)
	}
}

func (p *PriorityProvider) Load() error {
	for _, provider := range p.providers {
		if err := provider.Load(); err != nil {
			return fmt.Errorf("failed to load configuration with %T: %w", provider, err)
		}
	}

	for name, to := range p.m {
		if to.Empty() && slices.Contains(p.required, name) {
			p.missing = append(p.missing, name)
		}
	}

	return nil
}

func (p *PriorityProvider) Missing() []string {
	return p.missing
}
