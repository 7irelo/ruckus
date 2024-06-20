package safety

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	AllowLabelKey   = "ruckus.enabled"
	AllowLabelValue = "true"
)

const (
	DefaultDuration = 30 * time.Second
	DefaultInterval = 10 * time.Second
	MaxDuration     = 5 * time.Minute
)

var (
	ErrApplyRequired = errors.New("destructive execution requires --apply")
	ErrAckRequired   = errors.New("destructive execution requires --yes-i-understand")
)

func ValidateTarget(target string) error {
	if strings.TrimSpace(target) == "" {
		return errors.New("target is required")
	}
	return nil
}

func ValidateDuration(duration time.Duration, unsafeMaxDuration bool) error {
	if duration <= 0 {
		return errors.New("duration must be greater than 0")
	}
	if duration > MaxDuration && !unsafeMaxDuration {
		return fmt.Errorf("duration %s exceeds safety cap %s (pass --unsafe-max-duration to override)", duration, MaxDuration)
	}
	return nil
}

func ValidateInterval(interval time.Duration) error {
	if interval <= 0 {
		return errors.New("interval must be greater than 0")
	}
	return nil
}

func ValidateRunApproval(apply bool, yesIUnderstand bool) error {
	if !apply {
		return ErrApplyRequired
	}
	if !yesIUnderstand {
		return ErrAckRequired
	}
	return nil
}

func IsAllowlisted(labels map[string]string) bool {
	if len(labels) == 0 {
		return false
	}
	value, ok := labels[AllowLabelKey]
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(value), AllowLabelValue)
}
