package swarm

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"
)

// maxDialDialErrors is the maximum number of dial errors we record
const maxDialDialErrors = 16

// DialError is the error type returned when dialing.
type DialError struct {
	Peer       peer.ID
	DialErrors []TransportError
	Cause      error
	Skipped    int
}

func (e *DialError) Timeout() bool {
	return os.IsTimeout(e.Cause)
}

func (e *DialError) recordErr(addr ma.Multiaddr, err error) {
	if len(e.DialErrors) >= maxDialDialErrors {
		e.Skipped++
		return
	}
	e.DialErrors = append(e.DialErrors, TransportError{Address: addr, Cause: err})
}

func (e *DialError) Error() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "failed to dial %s:", e.Peer)
	if e.Cause != nil {
		fmt.Fprintf(&builder, " %s", e.Cause)
	}
	for _, te := range e.DialErrors {
		fmt.Fprintf(&builder, "\n  * [%s] %s", te.Address, te.Cause)
	}
	if e.Skipped > 0 {
		fmt.Fprintf(&builder, "\n    ... skipping %d errors ...", e.Skipped)
	}
	return builder.String()
}

func (e *DialError) Unwrap() error {
	if e == nil || len(e.DialErrors) == 0 {
		return nil
	}
	return chainError(e.DialErrors)
}

func (e *DialError) Is(target error) bool {
	if e == target {
		return true
	}
	return e != nil && e.Cause != nil && errors.Is(e.Cause, target)
}

var _ error = (*DialError)(nil)

// TransportError is the error returned when dialing a specific address.
type TransportError struct {
	Address ma.Multiaddr
	Cause   error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("failed to dial %s: %s", e.Address, e.Cause)
}

func (e *TransportError) Unwrap() error {
	return e.Cause
}

var _ error = (*TransportError)(nil)

// chainError is used to implement Unwrap for DialError
// errors.Is and errors.As only support `interface { Unwrap() []error }` from go1.20
//
// The implementation is a modified version of multierror.chain from:
// https://github.com/hashicorp/go-multierror/blob/main/multierror.go#L96
type chainError []TransportError

func (c chainError) Error() string {
	// The actual value is not important. Only want to implement error.
	if len(c) == 0 {
		return "chainError: []"
	}
	return c[0].Error()
}

func (c chainError) Unwrap() error {
	if len(c) == 1 {
		return nil
	}
	return c[1:]
}

func (c chainError) Is(target error) bool {
	if len(c) == 0 {
		return false
	}
	return errors.Is(&c[0], target)
}
