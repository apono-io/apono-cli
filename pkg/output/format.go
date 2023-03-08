package output

import "github.com/thediveo/enumflag"

type Format enumflag.Flag

const (
	Plain Format = iota
	JsonFormat
	YamlFormat
)

var FormatIds = map[Format][]string{
	JsonFormat: {"json"},
	YamlFormat: {"yaml"},
}

func FormatFlag(formatPtr *Format) *enumflag.EnumValue {
	return enumflag.New(formatPtr, "output", FormatIds, enumflag.EnumCaseSensitive)
}
