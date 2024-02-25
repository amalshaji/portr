package utils

import (
	"time"

	"github.com/briandowns/spinner"
)

func ShowLoading(text string, completed chan bool) {
	s := spinner.New(spinner.CharSets[39], 100*time.Millisecond)
	s.Prefix = text + " "

	go func() {
		s.Start()
		defer s.Stop()
		<-completed
	}()
}
