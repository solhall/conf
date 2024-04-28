package conf

import (
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

const tagName = "conf"

type Provider interface {
	// StringVar registers a pointer to a string that will be set to the value of
	// the configuration parameter with the given name. If the parameter is
	// not set, the fallback value will be used. If the parameter is
	// required and not set, it will be considered missing.
	StringVar(to *string, name, fallback string, required bool)
	IntVar(to *int, name string, fallback int, required bool)
	BoolVar(to *bool, name string, fallback bool, required bool)
	// Load loads the actual values into the pointers. It should be called
	// after all calls to Var.
	Load() error
	// Missing returns a list of missing required configuration parameters.
	// It must only be called after Load, as otherwise it will always be
	// empty.
	Missing() []string
}

// LoadFlags is a shorthand for using Load with the FlagProvider.
func LoadFlags(cfg any, args []string) error {
	return Load(cfg, WithProviders(NewFlagProvider(args)))
}

// LoadFlagsWithRemaining is a shorthand for using Load with the FlagProvider
// and a remaining function.
func LoadFlagsWithRemaining(cfg any, args []string, remainingFunc func([]string)) error {
	return Load(cfg, WithProviders(NewFlagProvider(args).WithRemainingFunc(remainingFunc)))
}

func LoadEnv(cfg any, getenv func(string) string) error {
	return Load(cfg, WithProviders(NewEnvProvider(getenv)))
}

type loadConfig struct {
	provider Provider

	remainingArgs *[]string
}

type LoadOption func(*loadConfig)

// WithProviders returns a LoadOption that sets multiple "providers" (sources
// of configuration values) to be used in order of priority. Later providers
// will override values from earlier providers. Only non-empty values will
// override.
func WithProviders(providers ...Provider) LoadOption {
	return func(c *loadConfig) {
		c.provider = NewPriorityProvider(providers...)
	}
}

// LoadAll is a shorthand for using Load with all available providers.
func LoadAll(cfg any) error {
	return Load(cfg, WithProviders(
		NewEnvProvider(os.Getenv),
		NewFlagProvider(os.Args[1:]),
	))
}

// Load loads configuration values into the given struct cfg. cfg must be a
// pointer to a struct.
// If no providers are given via the LoadOptions, the default provider is the
// environment, using os.Getenv.
func Load(cfg any, opts ...LoadOption) error {
	t := reflect.TypeOf(cfg)
	// cfg must be a pointer to a struct
	switch t.Kind() {
	case reflect.Ptr:
		t = t.Elem()
		if t.Kind() != reflect.Struct {
			return fmt.Errorf("expected pointer to struct, got pointer to %s", t.Kind())
		}
	default:
		return fmt.Errorf("expected pointer to struct, got %s", t.Kind())
	}

	c := &loadConfig{
		provider: NewEnvProvider(os.Getenv),
	}

	for _, opt := range opts {
		opt(c)
	}

	v := reflect.ValueOf(cfg).Elem()
	for i := 0; i < t.NumField(); i++ {
		if err := c.LoadField(t.Field(i), v.Field(i)); err != nil {
			return err
		}
	}

	if err := c.provider.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if missing := c.provider.Missing(); len(missing) > 0 {
		return fmt.Errorf("missing configuration parameters: %s", strings.Join(missing, ", "))
	}

	return nil
}

func (c *loadConfig) LoadField(field reflect.StructField, value reflect.Value) error {
	// if field is embedded struct, recursively load it
	if field.Type.Kind() == reflect.Struct {
		for i := 0; i < field.Type.NumField(); i++ {
			if err := c.LoadField(field.Type.Field(i), value.Field(i)); err != nil {
				return err
			}
		}
		return nil
	}

	// e.g. "field1,default=my value,required"
	tagVal := field.Tag.Get(tagName)
	if tagVal == "" {
		return nil
	}

	tagVal, required, fallback := parseTag(tagVal)

	switch field.Type.Kind() {
	case reflect.Int:
		var fallbackInt int
		if fallback != "" {
			var err error
			fallbackInt, err = strconv.Atoi(fallback)
			if err != nil {
				return fmt.Errorf("failed to parse fallback value %q as int: %w", fallback, err)
			}
		}
		c.provider.IntVar(value.Addr().Interface().(*int), tagVal, fallbackInt, required)
	case reflect.String:
		c.provider.StringVar(value.Addr().Interface().(*string), tagVal, fallback, required)
	case reflect.Bool:
		var fallbackBool bool
		if fallback != "" {
			var err error
			fallbackBool, err = strconv.ParseBool(fallback)
			if err != nil {
				return fmt.Errorf("failed to parse fallback value %q as bool: %w", fallback, err)
			}
		}
		c.provider.BoolVar(value.Addr().Interface().(*bool), tagVal, fallbackBool, required)
	}

	return nil
}

func parseTag(tag string) (name string, required bool, fallback string) {
	parts := strings.Split(tag, ",")
	name = parts[0]

	if slices.Contains(parts, "required") {
		required = true
	}

	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "default=") {
			fallback = strings.TrimPrefix(part, "default=")
		}
	}

	return name, required, fallback
}

type typ struct {
	kind      reflect.Kind
	intVal    *int
	stringVal *string
	boolVal   *bool
}

func (t typ) Empty() bool {
	switch t.kind {
	case reflect.String:
		return t.stringVal == nil || *t.stringVal == ""
	case reflect.Int:
		return t.intVal == nil
	case reflect.Bool:
		return t.boolVal == nil
	default:
		return true
	}
}
