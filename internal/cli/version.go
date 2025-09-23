package cli

import (
	"fmt"
	"github.com/gherlein/go-netgear/internal/types"
)

// VERSION will be set at compile time - see Github actions...
var VERSION = "dev"

type VersionCommand struct {
}

func (version *VersionCommand) Run(args *types.GlobalOptions) error {
	fmt.Println(VERSION)
	return nil
}
