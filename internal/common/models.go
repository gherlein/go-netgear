package common

import (
	"github.com/gherlein/go-netgear/internal/types"
)

func IsModel30x(nm types.NetgearModel) bool {
	return nm == types.GS305EP || nm == types.GS305EPP || nm == types.GS308EP || nm == types.GS308EPP || nm == types.GS30xEPx
}

func IsModel316(nm types.NetgearModel) bool {
	return nm == types.GS316EP || nm == types.GS316EPP
}

func IsSupportedModel(modelName string) bool {
	return IsModel30x(types.NetgearModel(modelName)) || IsModel316(types.NetgearModel(modelName))
}

func Filter[T any](ss []T, test func(T) bool) (ret []T) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}