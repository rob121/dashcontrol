// Command remote is a chromedp example demonstrating how to connect to an
// existing Chrome DevTools instance using a remote WebSocket URL.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

//scaling

var ctx context.Context
var chrome string
var port string
var nav string
var cmd *exec.Cmd
var verbose bool
var urlstr string
var d time.Duration
var refresh time.Duration
var scale int
var modkey string
var modifier input.Modifier
var zoomsteps []int = []int{25, 33, 50, 67, 75, 80, 90, 100, 110, 125, 150, 175, 200, 250, 300, 400, 500}
var xtool string
var unloader string = `var all = document.getElementsByTagName("*");
	for (var i=0, max=all.length; i < max; i++) {
		if(all[i].getAttribute("onbeforeunload")) {
			all[i].setAttribute("onbeforeunload", null);
		}
	}
	window.onbeforeunload = null;
   
    window.alert = function alert (message) {
        console.log (message);
    }
`

func main() {

	flag.IntVar(&scale, "Zoom", 100, "Zoom factor 100%")
	flag.StringVar(&xtool, "xtool", "/usr/bin/xdotool", "Xdo Tool Path")
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
			modkey = kb.Control
			modifier = input.ModifierCtrl
		case "darwin":
			chrome = `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`
			modkey = kb.Meta
			modifier = input.ModifierCommand
		case "linux":
			chrome = `/usr/bin/google-chrome`
			modkey = kb.Control
			modifier = input.ModifierCtrl
		}
	}

	started := make(chan bool)
	//"--profile-directory=None"
	cmd = exec.Command(chrome, "--disable-popup-blocking", "--disable-prompt-on-repost", "--ignore-profile-directory-if-not-exists", fmt.Sprintf("--remote-debugging-port=%s", port), "--start-fullscreen", "--kiosk", "--no-prompts", "--no-dialogs")
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

			SendEnter()

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

	ScaleTo(scale)

	go func() {

		t := time.NewTicker(refresh)

		for range t.C {
			SendEnter() //we send enter since chrome sometimes has pop up "are you sure" dialogs.
			chromedp.Run(ctx, chromedp.Reload(),
				chromedp.Evaluate(unloader, nil),
			)

		}

	}()

	select {}
}

func SendKey(keypress string) {

	parts := strings.Split(keypress, " ")

	cmd2 := exec.Command(xtool, parts...)
	cmd2.Env = os.Environ()
	cmd2.Env = append(cmd.Env, "DISPLAY=:0")

	go func() {
		stdout, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			fmt.Println("Send Key Error", err2.Error())
		}
		if len(string(stdout)) > 1 {
			fmt.Println(string(stdout))
		}
	}()
}

//Chrome scaling is by keyboard, to do 100% you go -7 down

func ScaleTo(to int) {

	//reset
	time.Sleep(100 * time.Millisecond)
	//this ensures we are at the bottom
	for i := 0; i < 20; i++ {
		//send 20
		SendKey("key Control_L+minus")
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)

	for _, v := range zoomsteps {

		if to <= v {
			break
		}
		SendKey("key Control_L+plus")
		time.Sleep(50 * time.Millisecond)
	}
}

func SendEnter() {

	SendKey("key Return")
}

func SendZoomIn() {
	SendKey("key Control_L+plus")
}

func SendZoomOut() {
	SendKey("key Control_L+minus")
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
		chromedp.Evaluate(unloader, nil),
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
	http.HandleFunc("/scaleup", httpScaleUpHandler)
	http.HandleFunc("/scale/{to}", httpScaleHandler)
	http.HandleFunc("/scaledown", httpScaleDownHandler)
	http.ListenAndServe(":3333", nil)
}

func httpScaleUpHandler(w http.ResponseWriter, r *http.Request) {

	SendZoomIn()
	fmt.Fprintf(w, "OK")

}

func httpScaleHandler(w http.ResponseWriter, r *http.Request) {

	to_raw := r.PathValue("to")
	to, _ := strconv.Atoi(to_raw)
	ScaleTo(to)
	fmt.Fprintf(w, "OK")

}

func httpScaleDownHandler(w http.ResponseWriter, r *http.Request) {

	SendZoomOut()
	fmt.Fprintf(w, "OK")

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

	SendEnter()

	if err := chromedp.Run(ctx,
		chromedp.Reload(),
	); err == nil {
		fmt.Fprintf(w, "OK")
	} else {
		fmt.Fprintf(w, err.Error())
	}
}
