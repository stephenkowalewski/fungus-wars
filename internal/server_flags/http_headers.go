package server_flags

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Header implements the type.Value interface for parsing flags that contain http headers
type Header http.Header

func (i *Header) String() string {
	return fmt.Sprintf("%v", http.Header(*i))
}

var validHeaderRegex = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]*:`)

func (i *Header) Set(value string) error {
	if !validHeaderRegex.MatchString(value) {
		return errors.New(`"` + value + `". Does not look like a valid HTTP Header`)
	}
	key_value := strings.SplitN(value, ":", 2)
	http.Header(*i).Set(key_value[0], strings.TrimSpace(key_value[1]))
	return nil
}
