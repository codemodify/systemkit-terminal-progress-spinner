package main

import (
	"time"

	spinner "github.com/codemodify/systemkit-terminal-progress-spinner"
)

func main() {
	successSpinner := spinner.NewSpinner("Running operation 1")
	successSpinner.Run()
	time.Sleep(5 * time.Second)
	successSpinner.Success()

	failSpinner := spinner.NewSpinner("Running operation 2")
	failSpinner.Run()
	time.Sleep(5 * time.Second)
	failSpinner.Fail()
}
