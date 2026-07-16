package main

import (
	"bufio"
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	puzzleURL    = "https://18words.com/"
	pollInterval = 25 * time.Millisecond
)

//go:embed disable-motion.js
var disableMotion string

func main() {
	verbose := flag.Bool("verbose", false, "show diagnostic logging")
	output := flag.String("output", "", "save a screenshot to this path")
	headful := flag.Bool("headful", false, "show the Chrome window while solving")
	flag.Parse()

	logger := log.New(io.Discard, "", log.LstdFlags)
	if *verbose {
		logger.SetOutput(os.Stderr)
	}

	elapsed, completed, err := solve(context.Background(), *output, *headful, logger)
	if !completed {
		fmt.Printf("Program time: %d ms\n", elapsed.Milliseconds())
	}
	if err != nil {
		log.Fatalf("solve puzzle: %v", err)
	}
}

func pollFor(expr string) chromedp.Action {
	return chromedp.Poll(expr, nil, chromedp.WithPollingInterval(pollInterval))
}

func waitForHeadfulClose(ctx context.Context) error {
	fmt.Print("Press Enter to close Chrome...")
	input := make(chan error, 1)
	go func() {
		_, err := bufio.NewReader(os.Stdin).ReadString('\n')
		input <- err
	}()

	select {
	case err := <-input:
		fmt.Println()
		if err == nil {
			return nil
		}
		if !errors.Is(err, io.EOF) {
			return fmt.Errorf("read terminal input: %w", err)
		}
		fmt.Println("No terminal input available; close Chrome to exit.")
		<-ctx.Done()
		return nil
	case <-ctx.Done():
		fmt.Println()
		return nil
	}
}

func solve(ctx context.Context, output string, headful bool, logger *log.Logger) (time.Duration, bool, error) {
	startedAt := time.Now()
	allocatorOptions := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", !headful),
	)
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	defer cancelAllocator()

	browserCtx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()
	actionCtx, cancelActions := context.WithTimeout(browserCtx, 45*time.Second)
	defer cancelActions()

	logger.Printf("opening %s", puzzleURL)
	if err := chromedp.Run(actionCtx,
		chromedp.Navigate(puzzleURL),
		chromedp.WaitVisible(`//button[normalize-space()="Play now"]`, chromedp.BySearch),
		chromedp.Evaluate(`document.getElementById("playBtn").click()`, nil),
	); err != nil {
		return time.Since(startedAt), false, fmt.Errorf("open game: %w", err)
	}
	if err := chromedp.Run(actionCtx,
		pollFor(`typeof state !== "undefined" && Array.isArray(state.words) && state.words.length > 0`),
		chromedp.Evaluate(disableMotion, nil),
	); err != nil {
		var diagnostics string
		_ = chromedp.Run(actionCtx, chromedp.Evaluate(
			`JSON.stringify({url: location.href, playButton: document.getElementById("playBtn")?.textContent, activeScreen: document.querySelector(".screen.active")?.id, hasState: typeof state !== "undefined", wordCount: typeof state !== "undefined" ? state.words?.length : null})`,
			&diagnostics,
		))
		return time.Since(startedAt), false, fmt.Errorf("start game (%s): %w", diagnostics, err)
	}

	var words []string
	if err := chromedp.Run(actionCtx, chromedp.Evaluate(`state.words`, &words)); err != nil {
		return time.Since(startedAt), false, fmt.Errorf("read answers: %w", err)
	}
	logger.Printf("loaded %d answers", len(words))

	for index, word := range words {
		if err := chromedp.Run(actionCtx, pollFor(
			fmt.Sprintf(`state.wordIdx === %d && !state.processing && state.selected.length === state.locked.length`, index),
		)); err != nil {
			return time.Since(startedAt), false, fmt.Errorf("wait for word %d: %w", index+1, err)
		}

		fmt.Printf("Solving %s\n", word)
		if err := chromedp.Run(actionCtx,
			chromedp.Evaluate(
				fmt.Sprintf(`Array.from(%q).forEach(key => document.dispatchEvent(new KeyboardEvent("keydown", {key, bubbles: true})))`, word),
				nil,
			),
			pollFor(fmt.Sprintf(`state.wordIdx > %d`, index)),
		); err != nil {
			return time.Since(startedAt), false, fmt.Errorf("solve word %d (%s): %w", index+1, word, err)
		}
	}

	if err := chromedp.Run(actionCtx, pollFor(
		`document.getElementById("result").classList.contains("active")`,
	)); err != nil {
		return time.Since(startedAt), false, fmt.Errorf("wait for result: %w", err)
	}
	solvedIn := time.Since(startedAt)
	fmt.Println("You won!")
	fmt.Printf("Program time: %d ms\n", solvedIn.Milliseconds())

	if output != "" {
		var screenshot []byte
		if err := chromedp.Run(actionCtx,
			pollFor(`document.getElementById("feedbackLine").classList.contains("show")`),
			chromedp.Sleep(100*time.Millisecond),
			chromedp.FullScreenshot(&screenshot, 100),
		); err != nil {
			return solvedIn, true, fmt.Errorf("capture screenshot: %w", err)
		}
		if err := os.WriteFile(output, screenshot, 0o644); err != nil {
			return solvedIn, true, fmt.Errorf("write screenshot: %w", err)
		}
		fmt.Printf("Screenshot saved to %s\n", output)
	}

	if headful {
		cancelActions()
		if err := waitForHeadfulClose(browserCtx); err != nil {
			return solvedIn, true, err
		}
	}
	return solvedIn, true, nil
}
