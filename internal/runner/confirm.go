package runner

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func (r *Runner) confirmTask(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(r.log.out, "%s [y/N]: ", strings.TrimSpace(prompt))
	line, _ := reader.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}
