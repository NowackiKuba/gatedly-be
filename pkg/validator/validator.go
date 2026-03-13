package validator

import (
	"regexp"
	"strings"
)

// slugRegex matches lowercase alphanumeric and hyphens: ^[a-z0-9]+(?:-[a-z0-9]+)*$
var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Validate accumulates field-level validation errors.
type Validate struct {
	Errors map[string]string
}

// New returns a new Validate.
func New() *Validate {
	return &Validate{Errors: make(map[string]string)}
}

// Check adds an error for field with msg if ok is false.
func (v *Validate) Check(ok bool, field, msg string) {
	if !ok {
		v.Errors[field] = msg
	}
}

// Valid returns true if there are no errors.
func (v *Validate) Valid() bool {
	return len(v.Errors) == 0
}

// Error implements error interface: joins all errors as "field: msg; field: msg".
func (v *Validate) Error() string {
	if len(v.Errors) == 0 {
		return ""
	}
	var parts []string
	for field, msg := range v.Errors {
		parts = append(parts, field+": "+msg)
	}
	return strings.Join(parts, "; ")
}

// IsSlug returns true if s matches ^[a-z0-9]+(?:-[a-z0-9]+)*$.
func IsSlug(s string) bool {
	return slugRegex.MatchString(s)
}
