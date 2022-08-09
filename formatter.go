package pine

import "fmt"

const (
	colorRed = iota + 31
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan

	colorBold     = 1
	colorDarkGray = 90
)

func colorize(s string, c int, useColors bool) string {
	if useColors {
		return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
	}
	return s
}
