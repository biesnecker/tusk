package output

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	infoColor    = color.New(color.FgCyan)
	urlColor     = color.New(color.FgBlue, color.Underline)
	promptColor  = color.New(color.FgYellow)
)

func Success(format string, a ...interface{}) {
	successColor.Printf("✓ "+format+"\n", a...)
}

func Error(format string, a ...interface{}) {
	errorColor.Printf("✗ "+format+"\n", a...)
}

func Info(format string, a ...interface{}) {
	infoColor.Printf(format+"\n", a...)
}

func URL(url string) {
	urlColor.Println(url)
}

func Prompt(format string, a ...interface{}) {
	promptColor.Printf(format, a...)
}

func Plain(format string, a ...interface{}) {
	fmt.Printf(format+"\n", a...)
}
