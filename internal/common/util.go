package common

import (
	"strconv"
)

func ParseFloat32(text string) float32 {
	i64, _ := strconv.ParseFloat(text, 32)
	return float32(i64)
}

func ParseInt32(text string) int32 {
	i64, _ := strconv.ParseInt(text, 10, 32)
	return int32(i64)
}