// Command remote is a chromedp example demonstrating how to connect to an
// existing Chrome DevTools instance using a remote WebSocket URL.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var ctx context.Context
var chrome string
var port string
var nav string
var cmd *exec.Cmd
var verbose bool
var urlstr string
var d time.Duration
var refresh time.Duration

func main() {

	flag.StringVar(&chrome, "chrome", "", "Chrome Path")
	flag.StringVar(&port, "port", "9222", "Chrome Port")
	flag.StringVar(&nav, "nav", "https://www.duckduckgo.com/", "nav")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.StringVar(&urlstr, "url", fmt.Sprintf("ws://127.0.0.1:%s", port), "devtools url")
	flag.DurationVar(&d, "d", 1*time.Second, "Wait Duration")
	flag.DurationVar(&refresh, "refresh", 60*time.Second, "Refresh Duration")
	flag.Parse()

	if len(chrome) < 1 {
		switch runtime.GOOS {
		case "windows":
			chrome = "chrome.exe"
		case "darwin":
			chrome = `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`
		case "linux":
			chrome = `/usr/bin/google-chrome`
		}
	}

	started := make(chan bool)

	cmd = exec.Command(chrome, "--profile-directory=Default", fmt.Sprintf("--remote-debugging-port=%s", port), "--start-fullscreen")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "DISPLAY=:0")

	go func() {
		err := cmd.Start()
		if err != nil {
			fmt.Println(err)

		}
		fmt.Println("PID", cmd.Process.Pid)
		time.Sleep(time.Second * 3)
		started <- true
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		for range c {
			// sig is a ^C, handle it
			//kill the cmd

			fmt.Println("PID", cmd.Process.Pid)

			if err := syscall.Kill(cmd.Process.Pid, syscall.SIGTERM); err != nil {
				fmt.Println("Error killing process:", err)
				os.Exit(1)
			}

			os.Exit(1)
		}
	}()

	fmt.Println("Waiting for ready")

	status := <-started

	go httpServer()

	fmt.Println("Ready?", status)

	if err := run(verbose, urlstr, nav, d); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	go func() {

		t := time.NewTicker(refresh)

		for range t.C {

			chromedp.Run(ctx, chromedp.Reload())

		}

	}()

	select {}
}

func run(verbose bool, urlstr, nav string, d time.Duration) error {
	if urlstr == "" {
		return errors.New("invalid remote devtools url")
	}
	// create allocator context for use with creating a browser context later
	allocatorContext, _ := chromedp.NewRemoteAllocator(context.Background(), urlstr)
	// defer cancel()

	// build context options
	var opts []chromedp.ContextOption
	if verbose {
		opts = append(opts, chromedp.WithDebugf(log.Printf))
	}

	// create context
	ctx, _ = chromedp.NewContext(allocatorContext, opts...)
	// defer cancel()

	// run task list
	if err := chromedp.Run(ctx,
		page.BringToFront(),
		chromedp.Navigate(nav),
	); err != nil {
		return fmt.Errorf("Failed getting body of %s: %v", nav, err)
	}

	cleanTabs()

	time.Sleep(time.Second * 10)

	return nil
}

func httpServer() {
	http.HandleFunc("/nav", httpNavigateHandler)
	http.HandleFunc("/refresh", httpRefreshHandler)
	http.ListenAndServe(":3333", nil)
}

func cleanTabs() {

	targets, _ := chromedp.Targets(ctx)

	for _, t := range targets {

		if t.URL == "chrome://newtab/" {
			tabCtx, _ := chromedp.NewContext(ctx, chromedp.WithTargetID(t.TargetID))
			if err := chromedp.Run(tabCtx, page.Close()); err != nil {
				log.Fatal(err)
			}
		}

	}

}

func httpNavigateHandler(w http.ResponseWriter, r *http.Request) {

	urlreq := r.URL.Query().Get("url")

	if err := chromedp.Run(ctx,
		page.BringToFront(),
		chromedp.Navigate(urlreq),
	); err == nil {
		fmt.Fprintf(w, "OK")
	} else {
		fmt.Fprintf(w, err.Error())
	}

	cleanTabs()
}

func httpRefreshHandler(w http.ResponseWriter, r *http.Request) {

	if err := chromedp.Run(ctx,
		chromedp.Reload(),
	); err == nil {
		fmt.Fprintf(w, "OK")
	} else {
		fmt.Fprintf(w, err.Error())
	}
}
