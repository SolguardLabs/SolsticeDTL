package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"solsticedtl/src/scenario"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "run":
		return runScenario(args[1:])
	case "validate":
		return validateScenario(args[1:])
	case "help", "-h", "--help":
		return usage()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runScenario(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("run requires a scenario file")
	}
	path := args[0]
	pretty := true
	includeEvents := false
	for _, arg := range args[1:] {
		switch arg {
		case "--json":
			pretty = false
		case "--pretty":
			pretty = true
		case "--events":
			includeEvents = true
		default:
			return fmt.Errorf("unknown run flag %q", arg)
		}
	}
	result, err := scenario.RunFile(path, includeEvents)
	if err != nil {
		return err
	}
	data, err := scenario.EncodeReport(result.Report, pretty)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func validateScenario(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("validate requires a scenario file")
	}
	messages, err := scenario.ValidateFile(args[0])
	if err != nil {
		return err
	}
	if hasJSON(args[1:]) {
		data, err := json.Marshal(map[string]any{"status": "ok", "messages": messages})
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	for _, message := range messages {
		fmt.Println(message)
	}
	return nil
}

func hasJSON(args []string) bool {
	for _, arg := range args {
		if strings.TrimSpace(arg) == "--json" {
			return true
		}
	}
	return false
}

func usage() error {
	fmt.Println("SolsticeDTL")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  solsticedtl run <scenario.json> [--json] [--events]")
	fmt.Println("  solsticedtl validate <scenario.json> [--json]")
	return nil
}
