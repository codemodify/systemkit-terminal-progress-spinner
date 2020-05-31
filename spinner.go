package progress

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	terminal "github.com/codemodify/systemkit-terminal"
	progress "github.com/codemodify/systemkit-terminal-progress"
)

// spinner -
type spinner struct {
	config progress.Config

	spinnerGlyphsIndex int

	stopChannel     chan bool
	stopWithSuccess bool
	finishedChannel chan bool

	lastPrintLen int

	theTerminal *terminal.Terminal
}

// NewSpinnerWithConfig -
func NewSpinnerWithConfig(config progress.Config) progress.Renderer {

	// 1. set defaults
	if config.Writer == nil {
		config.Writer = os.Stdout
	}

	// 2.
	return &spinner{
		config: config,

		spinnerGlyphsIndex: -1,

		stopChannel:     make(chan bool),
		stopWithSuccess: true,
		finishedChannel: make(chan bool),

		lastPrintLen: 0,

		theTerminal: terminal.NewTerminal(config.Writer),
	}
}

// NewSpinner -
func NewSpinner(args ...string) progress.Renderer {
	defaultConfig := progress.NewDefaultConfig(args...)
	defaultConfig.ProgressGlyphs = []string{"|", "/", "-", "\\"}

	return NewSpinnerWithConfig(*defaultConfig)
}

// Run -
func (thisRef *spinner) Run() {
	go thisRef.drawLineInLoop()
}

// Success -
func (thisRef *spinner) Success() {
	thisRef.stop(true)
}

// Fail -
func (thisRef *spinner) Fail() {
	thisRef.stop(false)
}

func (thisRef *spinner) stop(success bool) {
	thisRef.stopWithSuccess = success
	thisRef.stopChannel <- true
	close(thisRef.stopChannel)

	<-thisRef.finishedChannel
}

func (thisRef *spinner) drawLine(char string) (int, error) {
	return fmt.Fprintf(thisRef.config.Writer, "%s%s%s%s", thisRef.config.Prefix, char, thisRef.config.Suffix, thisRef.config.ProgressMessage)
}

func (thisRef *spinner) drawOperationProgressLine() {
	thisRef.spinnerGlyphsIndex++
	if thisRef.spinnerGlyphsIndex >= len(thisRef.config.ProgressGlyphs) {
		thisRef.spinnerGlyphsIndex = 0
	}

	if err := thisRef.eraseLine(); err != nil {
		return
	}

	n, err := thisRef.drawLine(thisRef.config.ProgressGlyphs[thisRef.spinnerGlyphsIndex])
	if err != nil {
		return
	}

	thisRef.lastPrintLen = n
}

func (thisRef *spinner) drawOperationStatusLine() {
	status := thisRef.config.SuccessGlyph
	if !thisRef.stopWithSuccess {
		status = thisRef.config.FailGlyph
	}

	if err := thisRef.eraseLine(); err != nil {
		return
	}

	if _, err := thisRef.drawLine(status); err != nil {
		return
	}

	fmt.Fprintf(thisRef.config.Writer, "\n")

	thisRef.lastPrintLen = 0
}

func (thisRef *spinner) drawLineInLoop() {
	if thisRef.config.HideCursor {
		thisRef.theTerminal.CursorHide()
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				thisRef.drawOperationProgressLine()

			case <-thisRef.stopChannel:
				ticker.Stop()
				return
			}
		}
	}()

	// for stop aka Success/Fail
	wg.Wait()

	thisRef.drawOperationStatusLine()

	if thisRef.config.HideCursor {
		thisRef.theTerminal.CursorShow()
	}

	thisRef.finishedChannel <- true
}

func (thisRef *spinner) eraseLine() error {
	_, err := fmt.Fprint(thisRef.config.Writer, "\r"+strings.Repeat(" ", thisRef.lastPrintLen)+"\r")
	return err
}
