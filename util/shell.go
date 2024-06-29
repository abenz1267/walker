package util

import (
	"strings"
)

func ParseShellCommand(command string) (string, []string) {
	f := strings.Fields(command)
	args := parseArgs(f)

	return args[0], args[1:]
}

func parseArgs(args []string) []string {
	if len(args) == 0 {
		return []string{}
	}

	arg := args[0]
	i := 1
	for ; i < len(args); i++ {
		startsWithQuote := (strings.HasPrefix(arg, "\"") || strings.HasPrefix(arg, "'")) && 
			!(strings.HasPrefix(arg, "\\\"") || strings.HasPrefix(arg, "\\'"))
	
		endsWithQuote := (strings.HasSuffix(arg, "\"") || strings.HasSuffix(arg, "'")) && 
			!(strings.HasSuffix(arg, "\\\"") || strings.HasSuffix(arg, "\\'"))
	
		endsWithBackslash := strings.HasSuffix(arg, "\\")

		if (!startsWithQuote || endsWithQuote) && !endsWithBackslash {
			break
		}

		arg += " " + args[i]
	}

	return append([]string{arg}, parseArgs(args[i:])...)
}