package dotenv_test

import (
	"io"
	"strings"
	"testing"

	"github.com/solhall/conf/dotenv"

	"github.com/google/go-cmp/cmp"
)

var simpleEnvFile = `SIMPLE_KEY=simple_value`

var commentedEnvFile = `#COMMENTED_KEY=commented_value`

var multilineEnvFile = `VALUE1=first_line
VALUE2=second_line`

var emptyLinesEnvFile = `

VALUE1=first_line

VALUE2=second_line
`

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		envFile string
		want    map[string]string
	}{
		{
			name:    "simple",
			envFile: simpleEnvFile,
			want: map[string]string{
				"SIMPLE_KEY": "simple_value",
			},
		},
		{
			name:    "commented",
			envFile: commentedEnvFile,
			want:    map[string]string{},
		},
		{
			name:    "multiline",
			envFile: multilineEnvFile,
			want: map[string]string{
				"VALUE1": "first_line",
				"VALUE2": "second_line",
			},
		},
		{
			name:    "empty_lines",
			envFile: emptyLinesEnvFile,
			want: map[string]string{
				"VALUE1": "first_line",
				"VALUE2": "second_line",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r io.Reader = strings.NewReader(tt.envFile)
			got, err := dotenv.ParseReader(r)
			if err != nil {
				t.Fatalf("unexpected error = %v", err)
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("unexpected env variables: %s", cmp.Diff(got, tt.want))
			}
		})
	}
}
