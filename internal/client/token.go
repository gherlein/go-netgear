package client

import (
	"fmt"
	"hash/adler32"
	"io"
	"os"
	"path/filepath"
	"github.com/gherlein/go-netgear/internal/common"
	"github.com/gherlein/go-netgear/internal/types"
)

const separator = ":"

func storeToken(args *types.GlobalOptions, host string, token string) error {
	err := ensureConfigPathExists(args.TokenDir)
	if err != nil {
		return err
	}
	if args.Verbose {
		fmt.Println("Storing login token " + tokenFilename(args.TokenDir, host))
	}
	data := fmt.Sprintf("%s%s%s", args.Model, separator, token)
	return os.WriteFile(tokenFilename(args.TokenDir, host), []byte(data), 0644)
}

func tokenFilename(configDir string, host string) string {
	hash32 := adler32.New()
	io.WriteString(hash32, host)
	return filepath.Join(dotConfigDirName(configDir), "token-"+fmt.Sprintf("%x", hash32.Sum(nil)))
}

func ReadTokenAndModel2GlobalOptions(args *types.GlobalOptions, host string) (types.NetgearModel, string, error) {
	return common.ReadTokenAndModel2GlobalOptions(args, host)
}

func ensureConfigPathExists(configDir string) error {
	dotConfigNtgrrc := dotConfigDirName(configDir)
	err := os.MkdirAll(dotConfigNtgrrc, os.ModeDir|0700)
	return err
}

func dotConfigDirName(configDir string) string {
	if configDir == "" {
		configDir = os.TempDir()
	}
	return filepath.Join(configDir, ".config", "ntgrrc")
}
