package types

import (
	"github.com/gherlein/go-netgear/internal/formatter"
)

type NetgearModel string

const (
	GS30xEPx NetgearModel = "GS30xEPx"
	GS305EP  NetgearModel = "GS305EP"
	GS305EPP NetgearModel = "GS305EPP"
	GS308EP  NetgearModel = "GS308EP"
	GS308EPP NetgearModel = "GS308EPP"
	GS316EP  NetgearModel = "GS316EP"
	GS316EPP NetgearModel = "GS316EPP"
)

type GlobalOptions struct {
	Verbose      bool
	Quiet        bool
	OutputFormat formatter.OutputFormat
	TokenDir     string
	Model        NetgearModel
	Token        string
}