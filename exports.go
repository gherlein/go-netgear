// Package go_netgear provides a Go library for controlling Netgear managed switches
package go_netgear

import (
	"github.com/gherlein/go-netgear/internal/cli"
	"github.com/gherlein/go-netgear/internal/client"
	"github.com/gherlein/go-netgear/internal/formatter"
	"github.com/gherlein/go-netgear/internal/models"
	"github.com/gherlein/go-netgear/internal/types"
)

// Export types
type GlobalOptions = types.GlobalOptions
type NetgearModel = types.NetgearModel
type OutputFormat = formatter.OutputFormat

// Export commands
type LoginCommand = client.LoginCommand
type PoeCommand = models.PoeCommand
type PortCommand = models.PortCommand
type DebugReportCommand = cli.DebugReportCommand
type VersionCommand = cli.VersionCommand
type HelpAllFlag = cli.HelpAllFlag

// Export POE sub-commands
type PoeStatusCommand = models.PoeStatusCommand
type PoeShowSettingsCommand = models.PoeShowSettingsCommand
type PoeSetConfigCommand = models.PoeSetConfigCommand
type PoeCyclePowerCommand = models.PoeCyclePowerCommand

// Export Port sub-commands
type PortSettingsCommand = models.PortSettingsCommand
type PortSetCommand = models.PortSetCommand

// Export data structures
type PoePortStatus = models.PoePortStatus
type PoePortSetting = models.PoePortSetting
type PortSetting = models.PortSetting

// Export constants
const (
	MarkdownFormat = formatter.MarkdownFormat
	JsonFormat     = formatter.JsonFormat
)

// Export model constants
const (
	GS30xEPx = types.GS30xEPx
	GS305EP  = types.GS305EP
	GS305EPP = types.GS305EPP
	GS308EP  = types.GS308EP
	GS308EPP = types.GS308EPP
	GS316EP  = types.GS316EP
	GS316EPP = types.GS316EPP
)

// Export version
var VERSION = cli.VERSION

// Export utility functions
var DetectNetgearModel = models.DetectNetgearModel