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

// Spinner -
type Spinner struct {
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
	return &Spinner{
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
	progressMessage := ""
	successMessage := ""
	failMessage := ""

	if len(args) > 0 {
		progressMessage = args[0]
	}

	if len(args) > 1 {
		successMessage = args[1]
	} else {
		successMessage = progressMessage
	}

	if len(args) > 2 {
		failMessage = args[2]
	} else {
		failMessage = progressMessage
	}

	return NewSpinnerWithConfig(progress.Config{
		Prefix:          "[",
		ProgressGlyphs:  []string{"|", "/", "-", "\\"},
		Suffix:          "] ",
		ProgressMessage: progressMessage,
		SuccessGlyph:    string('\u2713'), // check mark
		SuccessMessage:  successMessage,
		FailGlyph:       string('\u00D7'), // middle cross
		FailMessage:     failMessage,
		Writer:          os.Stdout,
		HideCursor:      true,
	})
}

// Run -
func (thisRef *Spinner) Run() {
	go thisRef.drawLineInLoop()
}

// Success -
func (thisRef *Spinner) Success() {
	thisRef.stop(true)
}

// Fail -
func (thisRef *Spinner) Fail() {
	thisRef.stop(false)
}

func (thisRef *Spinner) stop(success bool) {
	thisRef.stopWithSuccess = success
	thisRef.stopChannel <- true
	close(thisRef.stopChannel)

	<-thisRef.finishedChannel
}

func (thisRef *Spinner) drawLine(char string) (int, error) {
	return fmt.Fprintf(thisRef.config.Writer, "%s%s%s%s", thisRef.config.Prefix, char, thisRef.config.Suffix, thisRef.config.ProgressMessage)
}

func (thisRef *Spinner) drawOperationProgressLine() {
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

func (thisRef *Spinner) drawOperationStatusLine() {
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

func (thisRef *Spinner) drawLineInLoop() {
	if thisRef.config.HideCursor {
		thisRef.theTerminal.HideCursor()
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
		thisRef.theTerminal.ShowCursor()
	}

	thisRef.finishedChannel <- true
}

func (thisRef *Spinner) eraseLine() error {
	_, err := fmt.Fprint(thisRef.config.Writer, "\r"+strings.Repeat(" ", thisRef.lastPrintLen)+"\r")
	return err
}
