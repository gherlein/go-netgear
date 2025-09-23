package client

import (
	"errors"
	"strconv"
	"strings"
	"github.com/gherlein/go-netgear/internal/common"
	"github.com/gherlein/go-netgear/internal/types"
)

func max(a int, b int) int {
	if b > a {
		return b
	}
	return a
}

func suffixToLength(s string, length int) string {
	runeLength := len([]rune(s))
	if runeLength < length {
		diff := length - runeLength
		return s + strings.Repeat(" ", diff)
	}
	return s
}

func parseFloat32(text string) float32 {
	i64, _ := strconv.ParseFloat(text, 32)
	return float32(i64)
}

func parseInt32(text string) int32 {
	i64, _ := strconv.ParseInt(text, 10, 32)
	return int32(i64)
}

func ensureModelIs30x(args *types.GlobalOptions, host string) error {
	model, _, err := ReadTokenAndModel2GlobalOptions(args, host)
	if err != nil {
		return err
	}
	if !common.IsModel30x(model) {
		return errors.New("This command is not yet supported for your Netgear model. " +
			"You might want to support the project by creating an issue on Github")
	}
	return nil
}

func filter[T any](ss []T, test func(T) bool) (ret []T) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}
