package utils

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/pflag"
	"github.com/thediveo/enumflag"
	"gopkg.in/yaml.v3"
)

type Format enumflag.Flag

const (
	TableFormat Format = iota
	JSONFormat
	YamlFormat
)

func AddFormatFlag(flags *pflag.FlagSet, formatPtr *Format) {
	enumValue := enumflag.New(formatPtr, "output", formatIds, enumflag.EnumCaseSensitive)
	flags.VarP(enumValue, "output", "o", "Output format. Valid values are 'table', 'yaml', or 'json'")
}

func PrintObjectsAsJson(writer io.Writer, objects any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(objects)
}

func PrintObjectsAsYaml(writer io.Writer, objects any) error {
	bytes, err := yaml.Marshal(objects)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(writer, string(bytes))
	return err
}

var formatIds = map[Format][]string{
	TableFormat: {"table"},
	JSONFormat:  {"json"},
	YamlFormat:  {"yaml"},
}
