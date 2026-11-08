package color

import (
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	Header      = color.New(color.FgGreen, color.Bold)
	Bold        = color.New(color.Bold)
	Faint       = color.New(color.Faint)
	TCP         = color.New(color.FgCyan)
	UDP         = color.New(color.FgMagenta)
	LISTEN      = color.New(color.FgYellow)
	ESTABLISHED = color.New(color.FgGreen)
	Default     = color.New(color.FgWhite)
)

func Init(mode string) {
	switch mode {
	case "always":
		color.NoColor = false
	case "never":
		color.NoColor = true
	case "auto":
		if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
			color.NoColor = true
		} else {
			color.NoColor = false
		}
	}
}

func IsColorDisabled() bool {
	return color.NoColor
}

func GetProtoColor(proto string) *color.Color {
	switch strings.ToLower(proto) {
	case "tcp":
		return TCP
	case "udp":
		return UDP
	default:
		return Default
	}
}

func GetStateColor(state string) *color.Color {
	switch strings.ToUpper(state) {
	case "LISTEN", "LISTENING":
		return LISTEN
	case "ESTABLISHED":
		return ESTABLISHED
	default:
		return Default
	}
}
