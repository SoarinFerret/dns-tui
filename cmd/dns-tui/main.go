package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/soarinferret/dns-tui/internal/config"
	"github.com/soarinferret/dns-tui/internal/tui"
)

func main() {
	configPath := flag.String("config", config.DefaultPath(), "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) && !isNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		cfg = &config.Config{}
		fmt.Println("Welcome to dns-tui!")
		fmt.Println()
		profile, err := promptProfile()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		cfg.Profiles = append(cfg.Profiles, profile)
		if err := config.Save(*configPath, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nConfig saved to %s\n\n", *configPath)
	}

	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles configured. Let's add one.")
		fmt.Println()
		profile, err := promptProfile()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		cfg.Profiles = append(cfg.Profiles, profile)
		if err := config.Save(*configPath, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nConfig saved to %s\n\n", *configPath)
	}

	app := tui.New(cfg, *configPath)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func isNotExist(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such file or directory")
}

var providers = []struct {
	name   string
	fields []string
}{
	{"cloudflare", []string{"api_token"}},
	{"godaddy", []string{"api_key", "api_secret"}},
	{"dnsmadeeasy", []string{"api_key", "api_secret"}},
}

func promptProfile() (config.Profile, error) {
	reader := bufio.NewReader(os.Stdin)

	name, err := prompt(reader, "Profile name: ")
	if err != nil {
		return config.Profile{}, err
	}

	fmt.Println("Available providers:")
	for i, p := range providers {
		fmt.Printf("  %d) %s\n", i+1, p.name)
	}

	choiceStr, err := prompt(reader, "Select provider (1-3): ")
	if err != nil {
		return config.Profile{}, err
	}

	choice := 0
	if _, err := fmt.Sscanf(choiceStr, "%d", &choice); err != nil || choice < 1 || choice > len(providers) {
		return config.Profile{}, fmt.Errorf("invalid provider choice: %s", choiceStr)
	}

	selected := providers[choice-1]
	creds := make(map[string]string)
	for _, field := range selected.fields {
		val, err := prompt(reader, fmt.Sprintf("%s: ", field))
		if err != nil {
			return config.Profile{}, err
		}
		creds[field] = val
	}

	return config.Profile{
		Name:        name,
		Provider:    selected.name,
		Credentials: creds,
	}, nil
}

func prompt(reader *bufio.Reader, label string) (string, error) {
	fmt.Print(label)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
