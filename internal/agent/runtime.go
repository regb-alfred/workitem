package agent

import (
	"fmt"
	"strings"

	"github.com/regb/workitem/internal/model"
)

type Mode string

const (
	ModeTUI      Mode = "tui"
	ModeHeadless Mode = "headless"
)

func ParseMode(value string) (Mode, error) {
	mode := Mode(strings.ToLower(strings.TrimSpace(value)))
	switch mode {
	case ModeTUI, ModeHeadless:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid agent runtime mode %q; expected tui or headless", value)
	}
}

func (m Mode) Valid() bool {
	return m == ModeTUI || m == ModeHeadless
}

type Capabilities = model.RuntimeCapabilities

func CapabilitiesForMode(mode Mode) Capabilities {
	switch mode {
	case ModeTUI:
		return Capabilities{NativeTUI: true, Prompt: true, Streaming: true, Steering: true, FollowUp: true, Cancel: true, Monitor: true}
	case ModeHeadless:
		return Capabilities{Headless: true, Prompt: true, Streaming: true, Steering: true, FollowUp: true, Cancel: true, Monitor: true}
	default:
		return Capabilities{}
	}
}
