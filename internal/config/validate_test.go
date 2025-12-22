package config

import (
	"errors"
	"testing"
)

func TestValidateDefaults(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Errorf("Defaults().Validate() = %v", err)
	}
}

func TestValidateProblems(t *testing.T) {
	s := Defaults()
	s.Port = 0
	s.Output = "xml"
	err := s.Validate()
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err = %v, want *ValidationError", err)
	}
	keys := map[string]bool{}
	for _, p := range verr.Problems {
		keys[p.Key] = true
	}
	if !keys["port"] || !keys["output"] {
		t.Errorf("problems = %+v", verr.Problems)
	}
}

func TestValidateTimeouts(t *testing.T) {
	s := Defaults()
	s.ConnectTimeout = 0
	s.CommandTimeout = -1
	err := s.Validate()
	var verr *ValidationError
	if !errors.As(err, &verr) || len(verr.Problems) != 2 {
		t.Fatalf("err = %v, want two problems", err)
	}
}
