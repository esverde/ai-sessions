package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/esverde/ais/internal/config"
	"github.com/esverde/ais/internal/session"
	"github.com/esverde/ais/internal/ui"
)

const version = "0.1.0"

func main() {
	configOverride := flag.String("config", "", "configuration file path")
	providerOverride := flag.String("provider", "", "default provider override: all, claude, or codex")
	sortOverride := flag.String("sort", "", "default sort override: active or path")
	allProjects := flag.Bool("all", false, "start with all projects visible")
	initConfig := flag.Bool("init-config", false, "create the default configuration file and exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("ais", version)
		return
	}
	if *configOverride != "" {
		if err := os.Setenv("AIS_CONFIG", *configOverride); err != nil {
			fatal(err)
		}
	}
	cfg, configPath, err := config.Load()
	if err != nil {
		fatal(err)
	}
	if *initConfig {
		if err := config.Save(configPath, cfg); err != nil {
			fatal(err)
		}
		fmt.Println("Created", configPath)
		return
	}
	if *providerOverride != "" {
		cfg.Provider = *providerOverride
	}
	if *sortOverride != "" {
		cfg.Sort = *sortOverride
	}
	if *allProjects {
		cfg.Scope = config.ScopeAll
	}
	cfg = cfg.Normalize()

	cwd, err := session.CurrentDir()
	if err != nil {
		fatal(err)
	}
	model := ui.NewModel(cfg, configPath, cwd)
	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ais:", err)
	os.Exit(1)
}
