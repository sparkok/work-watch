package task

import (
	"bufio"
	"os"
	"strings"
)

// stdinReader is the shared stdin reader used by all interactive prompts.
var stdinReader = bufio.NewReader(os.Stdin)

// ReadLine reads one line from stdin and trims trailing \r\n.
// All interactive functions in work-watch MUST use this instead of creating
// their own bufio.Scanner or using fmt.Scanln, to avoid buffering conflicts.
func ReadLine() string {
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimRight(line, "\r\n")
}
