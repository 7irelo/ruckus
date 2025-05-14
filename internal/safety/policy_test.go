package safety

import (
	"errors"
	"testing"
	"time"
)

func TestValidateDurationSafetyCap(t *testing.T) {
	t.Parallel()

	if err := ValidateDuration(10*time.Second, false); err != nil {
		t.Fatalf("unexpected error for safe duration: %v", err)
	}

	if err := ValidateDuration(MaxDuration+time.Second, false); err == nil {
		t.Fatalf("expected error when duration exceeds cap without override")
	}

	if err := ValidateDuration(MaxDuration+time.Second, true); err != nil {
		t.Fatalf("expected override to allow longer duration, got: %v", err)
	}
}

func TestValidateRunApproval(t *testing.T) {
	t.Parallel()

	if err := ValidateRunApproval(true, true); err != nil {
		t.Fatalf("unexpected error for approved run: %v", err)
	}

	if err := ValidateRunApproval(false, true); !errors.Is(err, ErrApplyRequired) {
		t.Fatalf("expected ErrApplyRequired, got: %v", err)
	}

	if err := ValidateRunApproval(true, false); !errors.Is(err, ErrAckRequired) {
		t.Fatalf("expected ErrAckRequired, got: %v", err)
	}
}

func TestIsAllowlisted(t *testing.T) {
	t.Parallel()

	allowed := IsAllowlisted(map[string]string{AllowLabelKey: "true"})
	if !allowed {
		t.Fatalf("expected allowlisted labels to pass")
	}

	notAllowed := IsAllowlisted(map[string]string{AllowLabelKey: "false"})
	if notAllowed {
		t.Fatalf("expected false label value to be denied")
	}
}
