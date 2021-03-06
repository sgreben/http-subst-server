package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

type variables struct {
	Separator string

	Mu     sync.RWMutex
	Values map[string]string
	Texts  []string
}

func (fv *variables) help() string {
	separator := "="
	if fv.Separator != "" {
		separator = fv.Separator
	}
	return fmt.Sprintf("a variable definition NAME[%sVALUE] (if the value is omitted, the value of the environment variable with the given name is used)", separator)
}

// Set is flag.Value.Set
func (fv *variables) Set(v string) error {
	fv.Mu.Lock()
	defer fv.Mu.Unlock()
	separator := "="
	if fv.Separator != "" {
		separator = fv.Separator
	}
	if fv.Values == nil {
		fv.Values = make(map[string]string)
	}
	i := strings.Index(v, separator)
	var name, value string
	if i <= 0 {
		name = v
		value = os.Getenv(name)
	} else {
		name = v[:i]
		value = v[i+len(separator):]
	}
	fv.Texts = append(fv.Texts, v)
	fv.Values[name] = value
	return nil
}

func (fv *variables) String() string {
	fv.Mu.RLock()
	defer fv.Mu.RUnlock()
	return strings.Join(fv.Texts, ", ")
}
