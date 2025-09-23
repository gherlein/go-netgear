package common

import (
	"errors"
	"fmt"
	"hash/adler32"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"github.com/gherlein/go-netgear/internal/types"
)

const separator = ":"

func ReadTokenAndModel2GlobalOptions(args *types.GlobalOptions, host string) (types.NetgearModel, string, error) {

	if len(args.Model) > 0 && len(args.Token) > 0 {
		return args.Model, args.Token, nil
	}

	if args.Verbose {
		fmt.Println("reading token from: " + tokenFilename(args.TokenDir, host))
	}
	bytes, err := os.ReadFile(tokenFilename(args.TokenDir, host))
	if errors.Is(err, fs.ErrNotExist) {
		return "", "", errors.New("no session (token) exists. please login first")
	}
	data := strings.SplitN(string(bytes), separator, 2)
	if len(data) != 2 {
		return "", "", errors.New("you did an upgrade from a former ntgrcc version. please login again")
	}
	if !IsSupportedModel(data[0]) {
		return "", "", errors.New("unknown model stored in token. please login again")
	}
	args.Model = types.NetgearModel(data[0])
	args.Token = data[1]
	return args.Model, args.Token, err
}

func tokenFilename(configDir string, host string) string {
	hash32 := adler32.New()
	io.WriteString(hash32, host)
	return filepath.Join(dotConfigDirName(configDir), "token-"+fmt.Sprintf("%x", hash32.Sum(nil)))
}

func dotConfigDirName(configDir string) string {
	if configDir == "" {
		configDir = os.TempDir()
	}
	return filepath.Join(configDir, ".config", "ntgrrc")
}