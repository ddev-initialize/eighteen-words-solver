package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

const puzzleURL = "https://18words.com/"

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

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	elapsed, err := solve(ctx, *output, *headful, logger)
	fmt.Printf("Program time: %d ms\n", elapsed.Milliseconds())
	if err != nil {
		log.Fatalf("solve puzzle: %v", err)
	}
}

func solve(ctx context.Context, output string, headful bool, logger *log.Logger) (time.Duration, error) {
	startedAt := time.Now()
	allocatorOptions := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", !headful),
	)
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	defer cancelAllocator()

	browserCtx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	logger.Printf("opening %s", puzzleURL)
	if err := chromedp.Run(browserCtx,
		chromedp.Navigate(puzzleURL),
		chromedp.WaitVisible(`//button[normalize-space()="Play now"]`, chromedp.BySearch),
		chromedp.Evaluate(`document.getElementById("playBtn").click()`, nil),
	); err != nil {
		return time.Since(startedAt), fmt.Errorf("open game: %w", err)
	}
	if err := chromedp.Run(browserCtx,
		chromedp.Poll(
			`typeof state !== "undefined" && Array.isArray(state.words) && state.words.length > 0`,
			nil,
			chromedp.WithPollingInterval(50*time.Millisecond),
		),
		chromedp.Evaluate(disableMotion, nil),
	); err != nil {
		var diagnostics string
		_ = chromedp.Run(browserCtx, chromedp.Evaluate(
			`JSON.stringify({url: location.href, playButton: document.getElementById("playBtn")?.textContent, activeScreen: document.querySelector(".screen.active")?.id, hasState: typeof state !== "undefined", wordCount: typeof state !== "undefined" ? state.words?.length : null})`,
			&diagnostics,
		))
		return time.Since(startedAt), fmt.Errorf("start game (%s): %w", diagnostics, err)
	}

	var words []string
	if err := chromedp.Run(browserCtx, chromedp.Evaluate(`state.words`, &words)); err != nil {
		return time.Since(startedAt), fmt.Errorf("read answers: %w", err)
	}
	logger.Printf("loaded %d answers", len(words))

	for index, word := range words {
		if err := chromedp.Run(browserCtx, chromedp.Poll(
			fmt.Sprintf(`state.wordIdx === %d && !state.processing && state.selected.length === state.locked.length`, index),
			nil,
			chromedp.WithPollingInterval(25*time.Millisecond),
		)); err != nil {
			return time.Since(startedAt), fmt.Errorf("wait for word %d: %w", index+1, err)
		}

		fmt.Printf("Solving %s\n", word)
		if err := chromedp.Run(browserCtx,
			chromedp.Evaluate(
				fmt.Sprintf(`Array.from(%q).forEach(key => document.dispatchEvent(new KeyboardEvent("keydown", {key, bubbles: true})))`, word),
				nil,
			),
			chromedp.Poll(
				fmt.Sprintf(`state.wordIdx > %d`, index),
				nil,
				chromedp.WithPollingInterval(25*time.Millisecond),
			),
		); err != nil {
			return time.Since(startedAt), fmt.Errorf("solve word %d (%s): %w", index+1, word, err)
		}
	}

	if err := chromedp.Run(browserCtx, chromedp.Poll(
		`document.getElementById("result").classList.contains("active")`,
		nil,
		chromedp.WithPollingInterval(25*time.Millisecond),
	)); err != nil {
		return time.Since(startedAt), fmt.Errorf("wait for result: %w", err)
	}
	solvedIn := time.Since(startedAt)

	if output == "" {
		fmt.Println("You won!")
		return solvedIn, nil
	}

	var screenshot []byte
	if err := chromedp.Run(browserCtx,
		chromedp.Poll(
			`document.getElementById("feedbackLine").classList.contains("show")`,
			nil,
			chromedp.WithPollingInterval(25*time.Millisecond),
		),
		chromedp.Sleep(100*time.Millisecond),
		chromedp.FullScreenshot(&screenshot, 100),
	); err != nil {
		return solvedIn, fmt.Errorf("capture screenshot: %w", err)
	}
	if err := os.WriteFile(output, screenshot, 0o644); err != nil {
		return solvedIn, fmt.Errorf("write screenshot: %w", err)
	}

	fmt.Printf("You won! Screenshot saved to %s\n", output)
	return solvedIn, nil
}
