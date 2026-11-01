package executionusecase

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

const (
	// startCommandKind is the command kind that stores the identity of an interview start or a shared admit.
	startCommandKind = "start"
	// accessRevokedReason is the terminal reason stored when a start finds a revoked assignment.
	accessRevokedReason = "access_revoked"
	// maxInputRefs bounds the input references that take part in one intent.
	maxInputRefs = 16
)

// tokenPattern is the 1 to 64 character alphabet shared by scopes, keys, actors, reasons and references.
var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`)

// validToken reports whether value is an untrimmed token within the storage bounds.
func validToken(value string) bool { return tokenPattern.MatchString(value) }

// validUUID reports whether value parses as a UUID.
func validUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

// validRefs reports whether the input references fit the count and token bounds.
func validRefs(refs []string) bool {
	if len(refs) > maxInputRefs {
		return false
	}
	for _, ref := range refs {
		if !validToken(ref) {
			return false
		}
	}
	return true
}

// intentFingerprint hashes the scope, the binding being committed and the sorted unique input references.
// Every field is length-prefixed so adjacent values cannot collide.
func intentFingerprint(scope, instanceID, revisionID string, refs []string) string {
	unique := map[string]struct{}{}
	for _, ref := range refs {
		unique[ref] = struct{}{}
	}
	sorted := make([]string, 0, len(unique))
	for ref := range unique {
		sorted = append(sorted, ref)
	}
	sort.Strings(sorted)
	hash := sha256.New()
	write := func(value string) {
		hash.Write([]byte(strconv.Itoa(len(value))))
		hash.Write([]byte{':'})
		hash.Write([]byte(value))
	}
	for _, field := range append([]string{scope, instanceID, revisionID}, sorted...) {
		write(field)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// mintID returns the injected identity minter result, or a random UUID when none is injected.
func mintID(newID func() string) string {
	if newID != nil {
		return newID()
	}
	return uuid.NewString()
}

// trustedNow returns the clock instant as UTC at millisecond precision.
func trustedNow(clock service.Clock) time.Time { return clock.Now().UTC().Truncate(time.Millisecond) }

// mapStoreError converts a port failure to its public execution error or to the technical class.
func mapStoreError(err error) error {
	var coded *service.ExecutionError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &coded):
		return err
	case errors.Is(err, executionport.ErrNotFound):
		return service.ErrExecutionNotFound
	case errors.Is(err, executionport.ErrIdempotencyConflict):
		return service.ErrExecutionIdempotencyConflict
	case errors.Is(err, executionport.ErrUncertainCommit):
		return service.ErrExecutionUncertainCommit
	case errors.Is(err, executionport.ErrDependencyUnavailable):
		return service.ErrExecutionDependencyUnavailable
	case errors.Is(err, executionport.ErrAssignmentUnavailable):
		return service.ErrExecutionAssignmentUnavailable
	}
	return errs.Technical{Cause: err}
}
