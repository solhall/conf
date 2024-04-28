package dotenv

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func Parse() (map[string]string, error) {
	fname := ".env"
	f, err := os.Open(fname)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer f.Close()

	m, err := ParseReader(f)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", fname, err)
	}

	return m, nil
}

func ParseReader(r io.Reader) (map[string]string, error) {
	m := make(map[string]string)

	scanner := bufio.NewScanner(r)

	lineNr := 0

	for scanner.Scan() {
		lineNr++
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}

		if line == "" {
			continue
		}

		parts := strings.Split(line, "=")
		if len(parts) < 2 {
			return nil, fmt.Errorf("malformed line %d: %s", lineNr, line)
		}

		key := strings.TrimSpace(parts[0])

		var value string
		if len(parts) > 2 {
			value = strings.Join(parts[1:], "=")
		} else {
			value = strings.TrimSpace(parts[1])
		}

		m[key] = value
	}

	return m, nil
}
