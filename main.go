// Shutdown Timer schedules desktop power actions without root privileges.
package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

//go:embed icon.png
var iconData []byte

var version = "dev"

func main() {
	demo := flag.Bool("demo", false, "try the timer without running any power action")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("Shutdown Timer " + version)
		return
	}
	id := "io.github.cynd22.shutdown-timer"
	if *demo {
		id += ".demo"
	}
	a := app.NewWithID(id)
	a.SetIcon(fyne.NewStaticResource("shutdown-timer.png", iconData))
	w := a.NewWindow("Shutdown Timer")
	w.Resize(fyne.NewSize(480, 720))
	ui := newTimerUI(a, w, *demo, executeAction)
	ui.setupTray()
	w.SetCloseIntercept(ui.requestQuit)
	ctx, cancel := context.WithCancel(context.Background())
	w.SetOnClosed(cancel)
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fyne.Do(func() {
					if ctx.Err() == nil {
						ui.tick(time.Now())
					}
				})
			case <-ctx.Done():
				return
			}
		}
	}()
	w.ShowAndRun()
	cancel()
}
