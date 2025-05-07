package log

import (
	"fmt"
	"os"
	"strings"
)

const (
	symbolPos = 55
)

type Line interface {
	Success()
	Error(err error)
}

type logLine struct {
	textLength int
}

func Print(text string) *logLine {
	line := &logLine{
		textLength: len(text),
	}

	fmt.Print(text)
	return line
}

func (l *logLine) Warn(warn string) {
	l.addSpaces()
	fmt.Printf("\033[33m!\tWARN\033[0m %s\n", warn)
}

func (l *logLine) Success() {
	l.addSpaces()
	fmt.Println("\033[32m✔\033[0m")
}

func (l *logLine) Error(err error) {
	l.addSpaces()
	fmt.Println("\033[31mX\033[0m")
	fmt.Printf("\033[31mERROR\033[0m %s\n", err)
	os.Exit(-1)
}

func (l *logLine) addSpaces() {
	numOfSpaces := 1
	if l.textLength < symbolPos {
		numOfSpaces = symbolPos - l.textLength
	}

	fmt.Print(strings.Repeat(" ", numOfSpaces))
}
