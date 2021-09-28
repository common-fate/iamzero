package config

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// EnvFileParser is a FF parser for .env files
func EnvFileParser(prefix string) func(r io.Reader, set func(name, value string) error) error {
	return func(r io.Reader, set func(name, value string) error) error {

		s := bufio.NewScanner(r)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line == "" {
				continue // skip empties
			}

			if line[0] == '#' {
				continue // skip comments
			}

			var (
				name  string
				value string
				index = strings.IndexRune(line, '=')
			)
			if index < 0 {
				return errors.New("invalid .env file")
			} else {
				name, value = line[:index], strings.TrimSpace(line[index+1:])

				// PREFIX_MY_ENV_VAR -> MY_ENV_VAR
				name = maybeRemovePrefix(name, prefix)

				// MY_ENV_VAR -> my_env_var
				name = strings.ToLower(name)

				// my_env_var -> my-env-var
				name = envVarKeyReplacer.Replace(name)
			}

			if i := strings.Index(value, " #"); i >= 0 {
				value = strings.TrimSpace(value[:i])
			}

			if err := set(name, value); err != nil {
				return err
			}
		}
		return nil
	}
}

var envVarKeyReplacer = strings.NewReplacer(
	"_", "-",
)

func maybeRemovePrefix(key string, prefix string) string {
	if prefix == "" {
		return key
	}
	return strings.TrimPrefix(key, prefix+"_")
}
