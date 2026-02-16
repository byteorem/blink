// Package main is the entry point for the blink CLI.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/byteorem/blink/internal/config"
	"github.com/byteorem/blink/internal/copier"
	"github.com/byteorem/blink/internal/detect"
	"github.com/byteorem/blink/internal/ui"
	"github.com/byteorem/blink/internal/watcher"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

var version = "dev"

func main() {
	app := &cli.App{
		Name:    "blink",
		Usage:   "Hot-reload for WoW addon development",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "source",
				Aliases: []string{"s"},
				Usage:   "Path to addon source (default: auto-detect)",
			},
			&cli.StringFlag{
				Name:    "wow-path",
				Aliases: []string{"w"},
				Usage:   "Path to WoW version folder, e.g. /path/to/WoW/_retail_ (default: auto-detect)",
			},
			&cli.BoolFlag{
				Name:  "no-watch",
				Usage: "One-time copy, don't watch for changes",
			},
			&cli.IntFlag{
				Name:    "delay",
				Aliases: []string{"d"},
				Usage:   "Debounce delay in milliseconds (default: 50)",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose logging",
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	config.MergeFlags(&cfg, c.String("source"), c.String("wow-path"), c.Int("delay"), c.Bool("verbose"))

	if cfg.Verbose {
		log.Printf("[verbose] config: source=%q wowPath=%q delay=%dms gitignore=%v pkgmeta=%v ignore=%v",
			cfg.Source, cfg.WowPath, cfg.Delay, cfg.UseGitignore, cfg.UsePkgMeta, cfg.Ignore)
	}

	srcDir, addonName, err := detect.FindAddon(cfg.Source)
	if err != nil {
		return err
	}

	wowPath, err := detect.FindWowPath(cfg.WowPath)
	if err != nil {
		return err
	}

	if cfg.Verbose {
		log.Printf("[verbose] detected addon %q at %s", addonName, srcDir)
		log.Printf("[verbose] WoW path: %s", wowPath)
	}

	targetPath := filepath.Join(wowPath, "Interface", "AddOns", addonName)
	ig := copier.NewIgnorer(srcDir, cfg.Ignore, cfg.UseGitignore, cfg.UsePkgMeta)

	cleaned, err := copier.CleanDestination(srcDir, targetPath, ig)
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}
	if cleaned > 0 {
		fmt.Printf("Removed %d stale file(s) from destination\n", cleaned)
	}

	isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	var fileCount int

	if isTTY && !c.Bool("no-watch") {
		total, err := copier.CountFiles(srcDir, ig)
		if err != nil {
			return fmt.Errorf("counting files failed: %w", err)
		}

		syncModel := ui.NewSyncModel(total)
		p := tea.NewProgram(syncModel)

		go func() {
			_, _ = copier.InitialSyncWithProgress(srcDir, targetPath, ig, func(_ int) {
				p.Send(ui.SyncFileMsg{})
			})
			p.Send(ui.SyncDoneMsg{Count: total})
		}()

		if _, err := p.Run(); err != nil {
			return err
		}

		fileCount = total
	} else {
		var err error
		fileCount, err = copier.InitialSync(srcDir, targetPath, ig)
		if err != nil {
			return fmt.Errorf("initial sync failed: %w", err)
		}
	}

	if c.Bool("no-watch") {
		fmt.Printf("Synced %d files to %s\n", fileCount, targetPath)
		return nil
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	eventCh, err := watcher.Watch(ctx, srcDir, ig, cfg.Delay, cfg.Verbose)
	if err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	if isTTY {
		m := ui.NewModel(addonName, targetPath, srcDir, targetPath, fileCount, eventCh, ig)
		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			return err
		}
	} else {
		// Plain text mode for non-TTY
		fmt.Printf("blink %s — watching %s\n", version, addonName)
		fmt.Printf("target: %s\n", targetPath)
		fmt.Printf("synced %d files\n", fileCount)

		for ev := range eventCh {
			ts := time.Now().Format("15:04:05")

			if ev.Err != nil {
				fmt.Fprintf(os.Stderr, "%s  watcher error: %v\n", ts, ev.Err)
				continue
			}

			dstPath := filepath.Join(targetPath, ev.RelPath)
			srcPath := filepath.Join(srcDir, ev.RelPath)

			switch ev.Op {
			case watcher.OpRemove, watcher.OpRename:
				if err := copier.DeleteFile(dstPath); err != nil {
					fmt.Fprintf(os.Stderr, "%s  %s → error: %v\n", ts, ev.RelPath, err)
				} else {
					fmt.Printf("%s  %s → removed\n", ts, ev.RelPath)
				}
			default:
				if err := copier.CopyFile(srcPath, dstPath); err != nil {
					fmt.Fprintf(os.Stderr, "%s  %s → error: %v\n", ts, ev.RelPath, err)
				} else {
					fmt.Printf("%s  %s → copied\n", ts, ev.RelPath)
				}
			}
		}
	}

	return nil
}
