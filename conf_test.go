package conf_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/solhall/conf"
)

// don't want to use os.Setenv and os.Getenv in tests, since it's a global
// state
type env map[string]string

func (e env) Get(key string) string { return e[key] }

func TestLoad(t *testing.T) {
	t.Run("load string from env", func(t *testing.T) {
		type mystruct struct {
			Field1 string `conf:"field1,required"`
		}

		env := env{"FIELD1": "value1"}

		var cfg mystruct
		if err := conf.LoadEnv(&cfg, env.Get); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fmt.Printf("%+v\n", cfg)
		// Output:
		// Field1: value1
		if cfg.Field1 != "value1" {
			t.Fatalf("expected value %s, got %s", "value1", cfg.Field1)
		}
	})

	t.Run("load string from flags", func(t *testing.T) {
		type mystruct struct {
			Field1 string `conf:"field1,required"`
		}

		args := []string{"-field1", "value1"}

		var cfg mystruct
		if err := conf.LoadFlags(&cfg, args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fmt.Printf("%+v\n", cfg)
		// Output:
		// Field1: value1
		if cfg.Field1 != "value1" {
			t.Fatalf("expected value %s, got %s", "value1", cfg.Field1)
		}
	})

	t.Run("load into invalid type", func(t *testing.T) {
		type mystruct struct {
			Field1 *string `conf:"field1,required"`
		}

		var str string
		err := conf.Load(&str)
		if err.Error() != "expected pointer to struct, got pointer to string" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("load from strings from many", func(t *testing.T) {
		type mystruct struct {
			Field1 string `conf:"field1,required"`
			Field2 string `conf:"field2"`
			Field3 string `conf:"field3,default=defaultvalue"`
		}

		env := env{"FIELD1": "value1"}
		args := []string{"-field1", "value2"}

		var cfg mystruct
		if err := conf.Load(&cfg, conf.WithProviders(
			conf.NewEnvProvider(env.Get),
			conf.NewFlagProvider(args),
		)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fmt.Printf("%+v\n", cfg)
		// Output:
		// Field1: value2 (flag should override env due to order)
		if cfg.Field1 != "value2" {
			t.Fatalf("expected value %s, got %s", "value2", cfg.Field1)
		}

		if cfg.Field2 != "" {
			t.Fatalf("expected value %s, got %s", "", cfg.Field2)
		}

		if cfg.Field3 != "defaultvalue" {
			t.Fatalf("expected value %s, got %s", "defaultvalue", cfg.Field3)
		}
	})

	t.Run("load required int from env", func(t *testing.T) {
		type mystruct struct {
			Int1 int `conf:"int1,required"`
		}

		env := env{"INT1": "42"}

		var cfg mystruct
		if err := conf.LoadEnv(&cfg, env.Get); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fmt.Printf("%+v\n", cfg)
		// Output:
		// Int1: 42
		if cfg.Int1 != 42 {
			t.Fatalf("expected value %d, got %d", 42, cfg.Int1)
		}
	})

	t.Run("load bool from env", func(t *testing.T) {
		type mystruct struct {
			Bool1 bool `conf:"bool1"`
			Bool2 bool `conf:"bool2"`
		}

		env := env{"BOOL1": "true", "BOOL2": "1"}
		args := []string{}

		var cfg mystruct
		if err := conf.Load(&cfg, conf.WithProviders(
			conf.NewEnvProvider(env.Get),
			conf.NewFlagProvider(args),
		)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fmt.Printf("%+v\n", cfg)
		if cfg.Bool1 != true {
			t.Errorf("expected value %t, got %t", true, cfg.Bool1)
		}

		if cfg.Bool2 != true {
			t.Errorf("expected value %t, got %t", true, cfg.Bool2)
		}
	})

	t.Run("env upper snake case", func(t *testing.T) {
		type mystruct struct {
			Field1    string `conf:"field1,required"`
			FieldName string `conf:"field_name,required"`
		}

		env := env{"FIELD1": "value1", "FIELD_NAME": "value2"}

		var cfg mystruct
		if err := conf.LoadEnv(&cfg, env.Get); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fmt.Printf("%+v\n", cfg)
		// Output:
		// Field1: value1
		// FieldName: value2
		if cfg.Field1 != "value1" {
			t.Fatalf("expected value %s, got %s", "value1", cfg.Field1)
		}

		if cfg.FieldName != "value2" {
			t.Fatalf("expected value %s, got %s", "value2", cfg.FieldName)
		}
	})

	t.Run("flag lower case with hyphen", func(t *testing.T) {
		type mystruct struct {
			Field1    string `conf:"field1,required"`
			FieldName string `conf:"field_name,required"`
		}

		args := []string{"--field1", "value1", "--field-name", "value2"}

		var cfg mystruct
		if err := conf.LoadFlags(&cfg, args); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fmt.Printf("%+v\n", cfg)
		// Output:
		// Field1: value1
		// FieldName: value2
		if cfg.Field1 != "value1" {
			t.Fatalf("expected value %s, got %s", "value1", cfg.Field1)
		}

		if cfg.FieldName != "value2" {
			t.Fatalf("expected value %s, got %s", "value2", cfg.FieldName)
		}
	})

	t.Run("load realistic config", func(t *testing.T) {
		type mystruct struct {
			Host string `conf:"host,required"`
			Port int    `conf:"port,default=8080"`
			SSL  bool   `conf:"ssl,default=false"`

			// This field is not settable by the configuration
			// provider, so it should be ignored.
			ignoredField string

			// This field is settable, but has no tag.
			// It should be ignored.
			UntaggedField string

			// This field is required, but has a default value.
			// It should be set by the configuration provider.
			DefaultField string `conf:"default_field,required,default=defaultvalue"`

			// This field is required, but has no default value.
			// It should be set by the configuration provider.
			RequiredField string `conf:"required_field,required"`

			// This field is not required, and has no default value.
			// It should be left as the zero value.
			OptionalField string `conf:"optional_field"`

			// This field is not required, and has a default value.
			// It should be set by the configuration provider.
			OptionalDefaultField string `conf:"optional_default_field,default=default value"`
		}

		env := env{"HOST": "example.com"}
		args := []string{"--required-field", "value1"}

		var cfg mystruct
		if err := conf.Load(&cfg, conf.WithProviders(
			conf.NewEnvProvider(env.Get),
			conf.NewFlagProvider(args),
		)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fmt.Printf("%+v\n", cfg)
		// Output:
		// Host: example.com
		// Port: 8080
		// SSL: false
		// DefaultField: defaultvalue
		// RequiredField: value1
		// OptionalField:
		// OptionalDefaultField: default value
		// ignoredField:
		if cfg.Host != "example.com" {
			t.Fatalf("expected value %s, got %s", "example.com", cfg.Host)
		}

		if cfg.Port != 8080 {
			t.Fatalf("expected value %d, got %d", 8080, cfg.Port)
		}

		if cfg.SSL != false {
			t.Fatalf("expected value %t, got %t", false, cfg.SSL)
		}

		if cfg.DefaultField != "defaultvalue" {
			t.Fatalf("expected value %s, got %s", "defaultvalue", cfg.DefaultField)
		}

		if cfg.RequiredField != "value1" {
			t.Fatalf("expected value %s, got %s", "value1", cfg.RequiredField)
		}

		if cfg.OptionalField != "" {
			t.Fatalf("expected value %s, got %s", "", cfg.OptionalField)
		}

		if cfg.OptionalDefaultField != "default value" {
			t.Fatalf("expected value %s, got %s", "default value", cfg.OptionalDefaultField)
		}

		if cfg.ignoredField != "" {
			t.Fatalf("expected value %s, got %s", "", cfg.ignoredField)
		}

		if cfg.UntaggedField != "" {
			t.Fatalf("expected value %s, got %s", "", cfg.UntaggedField)
		}
	})

	t.Run("load flags until subcommand", func(t *testing.T) {
		type mystruct struct {
			Field1 string `conf:"field1,required"`
		}

		args := []string{"--field1", "value1", "subcommand", "--field1", "value2"}

		var cfg mystruct
		if err := conf.LoadFlagsWithRemaining(&cfg, args, func(remaining []string) {
			args = remaining
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fmt.Printf("%+v\n", cfg)
		// Output:
		// Field1: value1
		if cfg.Field1 != "value1" {
			t.Fatalf("expected value %s, got %s", "value1", cfg.Field1)
		}

		if len(args) != 3 || args[0] != "subcommand" || args[1] != "--field1" || args[2] != "value2" {
			t.Fatalf("expected remaining args %v, got %v", []string{"subcommand", "--field1"}, args)
		}
	})

	t.Run("load env with dotenv", func(t *testing.T) {
		type mystruct struct {
			OverrideMe  string `conf:"override-me"`
			NotInDotEnv string `conf:"not-in-dotenv"`
			Field2      string `conf:"field2"`
		}

		dotenv := "OVERRIDE_ME=overridden\nFIELD2=from-dotenv"
		r := strings.NewReader(dotenv)
		var rc io.ReadCloser = ioutil.NopCloser(r)

		env := env{"NOT_IN_DOTENV": "not-in-dotenv"}

		var cfg mystruct
		if err := conf.Load(&cfg, conf.WithProviders(
			conf.NewEnvProvider(env.Get).WithDotEnv(rc),
		)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fmt.Printf("%+v\n", cfg)
		// Output:
		// OverrideMe: overridden
		// NotInDotEnv: not-in-dotenv
		// Field2: from-dotenv
		if cfg.OverrideMe != "overridden" {
			t.Fatalf("expected value %s, got %s", "overridden", cfg.OverrideMe)
		}

		if cfg.NotInDotEnv != "not-in-dotenv" {
			t.Fatalf("expected value %s, got %s", "not-in-dotenv", cfg.NotInDotEnv)
		}

		if cfg.Field2 != "from-dotenv" {
			t.Fatalf("expected value %s, got %s", "from-dotenv", cfg.Field2)
		}
	})
}
