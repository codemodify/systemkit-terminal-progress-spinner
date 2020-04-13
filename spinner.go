package progress

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

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
func (s *Spinner) Run() {
	go s.drawLineInLoop()
}

// Success -
func (s *Spinner) Success() {
	s.stop(true)
}

// Fail -
func (s *Spinner) Fail() {
	s.stop(false)
}

func (s *Spinner) stop(success bool) {
	s.stopWithSuccess = success
	s.stopChannel <- true
	close(s.stopChannel)

	<-s.finishedChannel
}

func (s *Spinner) drawLine(char string) (int, error) {
	return fmt.Fprintf(s.config.Writer, "%s%s%s%s", s.config.Prefix, char, s.config.Suffix, s.config.ProgressMessage)
}

func (s *Spinner) drawOperationProgressLine() {
	s.spinnerGlyphsIndex++
	if s.spinnerGlyphsIndex >= len(s.config.ProgressGlyphs) {
		s.spinnerGlyphsIndex = 0
	}

	if err := s.eraseLine(); err != nil {
		return
	}

	n, err := s.drawLine(s.config.ProgressGlyphs[s.spinnerGlyphsIndex])
	if err != nil {
		return
	}

	s.lastPrintLen = n
}

func (s *Spinner) drawOperationStatusLine() {
	status := s.config.SuccessGlyph
	if !s.stopWithSuccess {
		status = s.config.FailGlyph
	}

	if err := s.eraseLine(); err != nil {
		return
	}

	if _, err := s.drawLine(status); err != nil {
		return
	}

	fmt.Fprintf(s.config.Writer, "\n")

	s.lastPrintLen = 0
}

func (s *Spinner) drawLineInLoop() {
	s.hideCursor()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				s.drawOperationProgressLine()

			case <-s.stopChannel:
				ticker.Stop()
				return
			}
		}
	}()

	// for stop aka Success/Fail
	wg.Wait()

	s.drawOperationStatusLine()

	s.unhideCursor()

	s.finishedChannel <- true
}

func (s *Spinner) eraseLine() error {
	_, err := fmt.Fprint(s.config.Writer, "\r"+strings.Repeat(" ", s.lastPrintLen)+"\r")
	return err
}

func (s *Spinner) hideCursor() error {
	if !s.config.HideCursor {
		return nil
	}

	_, err := fmt.Fprint(s.config.Writer, "\r\033[?25l\r")
	return err
}

func (s *Spinner) unhideCursor() error {
	if !s.config.HideCursor {
		return nil
	}

	_, err := fmt.Fprint(s.config.Writer, "\r\033[?25h\r")
	return err
}
