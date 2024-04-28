package conf

import (
	"flag"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
)

var _ Provider = (*FlagProvider)(nil)

type FlagProvider struct {
	m        map[string]typ
	fs       *flag.FlagSet
	required []string
	missing  []string
	args     []string

	remainingFunc func(remaining []string)
}

// NewFlagProvider creates a new FlagProvider that reads flags into `conf`
// tags.
// The remainingFunc is called with the remaining arguments, useful for
// subcommands. Can be nil.
func NewFlagProvider(args []string) *FlagProvider {
	fs := flag.NewFlagSet("confflags", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	return &FlagProvider{
		m:       make(map[string]typ),
		fs:      fs,
		missing: []string{},
		args:    args,
	}
}

func (p *FlagProvider) WithRemainingFunc(f func(remaining []string)) *FlagProvider {
	p.remainingFunc = f
	return p
}

func (p *FlagProvider) StringVar(to *string, name, fallback string, required bool) {
	name = p.normalizeName(name)

	p.m[name] = typ{kind: reflect.String, stringVal: to}
	p.fs.StringVar(to, name, fallback, "")
	if required {
		p.required = append(p.required, name)
	}
}

func (p *FlagProvider) IntVar(to *int, name string, fallback int, required bool) {
	name = p.normalizeName(name)

	p.m[name] = typ{kind: reflect.Int, intVal: to}
	p.fs.IntVar(to, name, fallback, "")
	if required {
		p.required = append(p.required, name)
	}
}

func (p *FlagProvider) BoolVar(to *bool, name string, fallback bool, required bool) {
	name = p.normalizeName(name)

	p.m[name] = typ{kind: reflect.Bool, boolVal: to}
	p.fs.BoolVar(to, name, fallback, "")
	if required {
		p.required = append(p.required, name)
	}
}

func (p *FlagProvider) Load() error {
	if err := p.fs.Parse(p.args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	for name, to := range p.m {
		if to.Empty() && slices.Contains(p.required, name) {
			p.missing = append(p.missing, name)
		}
	}

	// Pass remaining arguments to the remainingFunc
	if p.remainingFunc != nil {
		p.remainingFunc(p.fs.Args())
	}

	return nil
}

func (p *FlagProvider) Missing() []string {
	return p.missing
}

func (p *FlagProvider) normalizeName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}
