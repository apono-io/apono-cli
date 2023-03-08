package output

import (
	"github.com/spf13/pflag"
	"github.com/thediveo/enumflag"
)

type Format enumflag.Flag

const (
	Plain Format = iota
	JSONFormat
	YamlFormat
)

func AddFormatFlag(flags *pflag.FlagSet, formatPtr *Format) {
	enumValue := enumflag.New(formatPtr, "output", formatIds, enumflag.EnumCaseSensitive)
	flags.VarP(enumValue, "output", "o", "One of 'yaml' or 'json'")
}

var formatIds = map[Format][]string{
	JSONFormat: {"json"},
	YamlFormat: {"yaml"},
}
