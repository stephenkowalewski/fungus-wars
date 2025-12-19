package server_flags

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// Logfile implements the flag.Value interface to parse logfile args
type Logfile struct {
	Logger **log.Logger
	Name   string
}

func (v *Logfile) String() string {
	if v.Logger == nil {
		return "Logger(nil)"
	} else if v.Name == "" {
		return "Logger(no name)"
	}
	return v.Name
}

// Set opens a Logger based on a flag. File descriptors can be given
// in the form "fd:N". Otherwise the argument is presumed to be a file
// path with an optional "file:" prefix
func (v *Logfile) Set(s string) error {
	var f *os.File
	var err error
	v.Name = s

	if fArg, found := strings.CutPrefix(s, "fd:"); found {
		fd, err := strconv.Atoi(fArg)
		if err != nil {
			return errors.New(`Could not parse file descriptor "` + fArg + `" as an integar`)
		}
		f = os.NewFile(uintptr(fd), s)
		if f == nil {
			return errors.New("Unable to open file descriptor " + fArg)
		}
		if _, err = f.Stat(); err != nil {
			return fmt.Errorf("file descriptor %d is invalid: stat failed: %v", fd, err)
		}
	} else {
		fArg, _ := strings.CutPrefix(s, "file:")
		f, err = os.OpenFile(fArg, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
	}

	(*v.Logger).SetOutput(f)
	return nil
}
