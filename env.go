package conf

import (
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/solhall/conf/dotenv"
)

func NewEnvProvider(getenv func(string) string) *EnvProvider {
	return &EnvProvider{
		getenv:    getenv,
		m:         make(map[string]typ),
		fallbacks: make(map[string]string),
		missing:   []string{},
	}
}

func (p *EnvProvider) WithDotEnv(r io.ReadCloser) *EnvProvider {
	p.withDotEnv = true
	if r != nil {
		p.dotEnvReader = r
	}

	return p
}

type EnvProvider struct {
	withDotEnv   bool
	dotEnvReader io.ReadCloser

	getenv    func(string) string
	m         map[string]typ
	fallbacks map[string]string
	required  []string
	// missing is a list of missing required configuration parameters. If a
	// parameter does not have a value after considering the fallbacks map
	// and it is required, it will be considered missing.
	missing []string
}

var _ Provider = (*EnvProvider)(nil)

// Load reads the configuration from the environment variables.
func (p *EnvProvider) Load() error {
	if p.withDotEnv {
		if err := p.AddDotEnv(); err != nil {
			return fmt.Errorf("failed to add .env file: %w", err)
		}
	}

	getenv := func(name string) string {
		uppersnake := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
		return p.getenv(uppersnake)
	}

	for name, to := range p.m {
		rawVal := getenv(name)
		if rawVal == "" {
			if fallback, ok := p.fallbacks[name]; ok {
				rawVal = fallback
			}
		}

		if rawVal == "" {
			if slices.Contains(p.required, name) {
				p.missing = append(p.missing, name)
			}

			continue
		}

		switch to.kind {
		case reflect.String:
			*to.stringVal = rawVal
		case reflect.Int:
			val, err := strconv.Atoi(rawVal)
			if err != nil {
				return fmt.Errorf("failed to parse %s as int: %w", name, err)
			}
			*to.intVal = val
		case reflect.Bool:
			val, err := strconv.ParseBool(rawVal)
			if err != nil {
				return fmt.Errorf("failed to parse %s as bool: %w", name, err)
			}
			*to.boolVal = val
		default:
			return fmt.Errorf("field %s has unsupported type %s", name, to.kind)
		}
	}

	return nil
}

func (p *EnvProvider) AddDotEnv() error {
	var dotenvs map[string]string
	var err error
	if p.dotEnvReader != nil {
		dotenvs, err = dotenv.ParseReader(p.dotEnvReader)
		if err != nil {
			return fmt.Errorf("failed to parse .env file: %w", err)
		}
		if err := p.dotEnvReader.Close(); err != nil {
			return fmt.Errorf("failed to close .env file: %w", err)
		}
	} else {
		dotenvs, err = dotenv.Parse()
		if err != nil {
			return fmt.Errorf("failed to parse .env file: %w", err)
		}
	}

	defaultGetenv := p.getenv
	p.getenv = func(name string) string {
		if v, ok := dotenvs[name]; ok {
			return v
		}

		return defaultGetenv(name)
	}

	return nil
}

func (p *EnvProvider) StringVar(to *string, name, fallback string, required bool) {
	p.m[name] = typ{kind: reflect.String, stringVal: to}
	if fallback != "" {
		p.fallbacks[name] = fallback
	}
	if required {
		p.required = append(p.required, name)
	}
}

func (p *EnvProvider) IntVar(to *int, name string, fallback int, required bool) {
	p.m[name] = typ{kind: reflect.Int, intVal: to}
	if fallback != 0 {
		p.fallbacks[name] = fmt.Sprintf("%d", fallback)
	}
	if required {
		p.required = append(p.required, name)
	}
}

func (p *EnvProvider) BoolVar(to *bool, name string, fallback bool, required bool) {
	p.m[name] = typ{kind: reflect.Bool, boolVal: to}
	if fallback {
		p.fallbacks[name] = "true"
	} else {
		p.fallbacks[name] = "false"
	}
	if required {
		p.required = append(p.required, name)
	}
}

func (p *EnvProvider) Missing() []string {
	return p.missing
}
