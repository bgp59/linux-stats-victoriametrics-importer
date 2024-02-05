// Command line parsing utilities:

package lsvmi

import (
	"bytes"
	"flag"
	"strconv"
	"strings"
)

const (
	// The help usage message line wraparound default width:
	DEFAULT_FLAG_USAGE_WIDTH = 58
)

// Variant of flags which allows checking if they were was set or not; this is
// needed for command line args that override a setting, but *only* if they were
// used on the command line.

type BoolFlagCheckUsed struct {
	// Whether it was used on the command line or not:
	Used bool
	// Value, defaults to true if if no value was set on the command line:
	Value bool
}

func (bfcu *BoolFlagCheckUsed) BoolFlagFunc(s string) error {
	var (
		val bool
		err error
	)
	if s != "" {
		val, err = strconv.ParseBool(s)
		if err != nil {
			return err
		}
	} else {
		val = true
	}
	bfcu.Used = true
	bfcu.Value = val
	return nil
}

func NewBoolFlagCheckUsed(name, usage string) *BoolFlagCheckUsed {
	newBfcu := &BoolFlagCheckUsed{}
	flag.BoolFunc(
		name,
		FormatFlagUsage(usage),
		newBfcu.BoolFlagFunc,
	)
	return newBfcu
}

type StringFlagCheckUsed struct {
	// Whether it was used on the command line or not:
	Used bool
	// Value, populated w/ the default:
	Value string
}

func (sfcu *StringFlagCheckUsed) StringFlagFunc(s string) error {
	sfcu.Used = true
	sfcu.Value = s
	return nil
}

func NewStringFlagCheckUsed(name, value, usage string) *StringFlagCheckUsed {
	newSfcu := &StringFlagCheckUsed{
		Value: value,
	}
	flag.Func(
		name,
		FormatFlagUsage(usage),
		newSfcu.StringFlagFunc,
	)
	return newSfcu
}

// Format command flag usage for help message, by wrapping the lines around a
// given width. The original line breaks and prefixing white spaces are ignored.
// Example:
//
// var  flagArg = flag.String(
//
//	name,
//	value,
//	FormatFlagUsageWidth(`
//	This usage message will be reformatted to the given width, discarding
//	the current line breaks and line prefixing spaces.
//	`, 40),
//
// )
func FormatFlagUsageWidth(usage string, width int) string {
	buf := &bytes.Buffer{}
	lineLen := 0
	for i, word := range strings.Fields(strings.TrimSpace(usage)) {
		// Perform line length checking only if this is not the 1st word:
		if i > 0 {
			if lineLen+len(word)+1 > width {
				buf.WriteByte('\n')
				lineLen = 0
			} else {
				buf.WriteByte(' ')
				lineLen++
			}
		}
		n, err := buf.WriteString(word)
		if err != nil {
			return usage
		}
		lineLen += n
	}
	return buf.String()
}

func FormatFlagUsage(usage string) string {
	return FormatFlagUsageWidth(usage, DEFAULT_FLAG_USAGE_WIDTH)
}
