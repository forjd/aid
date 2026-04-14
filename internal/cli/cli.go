package cli

import (
	"fmt"
	"io"

	"aid/internal/output"
)

type Streams struct {
	Out     io.Writer
	Err     io.Writer
	Options output.Options
}

type Command struct {
	Name        string
	Path        string
	Use         string
	Usage       string
	Summary     string
	Description string
	Examples    []string
	Children    []*Command
	Run         func(args []string, streams Streams) error
}

func Run(args []string, stdout, stderr io.Writer) int {
	options, filteredArgs, err := parseGlobalOptions(args)
	if err != nil {
		if options.IsJSON() {
			_ = output.WriteError(stdout, inferCommandPath(rootCommand(), filteredArgs), err)
			return 1
		}
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	streams := Streams{
		Out:     stdout,
		Err:     stderr,
		Options: options,
	}

	root := rootCommand()
	if err := dispatch(root, filteredArgs, streams); err != nil {
		if streams.Options.IsJSON() {
			_ = output.WriteError(stdout, inferCommandPath(root, filteredArgs), err)
			return 1
		}

		fmt.Fprintf(streams.Err, "Error: %v\n", err)
		return 1
	}

	return 0
}

func dispatch(cmd *Command, args []string, streams Streams) error {
	if len(args) == 0 {
		if cmd.Run != nil {
			return cmd.Run(nil, streams)
		}

		renderHelp(cmd, streams.Out)
		return nil
	}

	if isHelpFlag(args[0]) {
		renderHelp(cmd, streams.Out)
		return nil
	}

	if len(cmd.Children) > 0 {
		if args[0] == "help" {
			target, err := lookupHelpTarget(cmd, args[1:])
			if err != nil {
				return err
			}

			renderHelp(target, streams.Out)
			return nil
		}

		if child := cmd.child(args[0]); child != nil {
			return dispatch(child, args[1:], streams)
		}
	}

	if cmd.Run != nil {
		return cmd.Run(args, streams)
	}

	return fmt.Errorf("unknown command %q", args[0])
}

func lookupHelpTarget(start *Command, path []string) (*Command, error) {
	current := start

	for _, part := range path {
		if isHelpFlag(part) {
			break
		}

		child := current.child(part)
		if child == nil {
			return nil, fmt.Errorf("unknown command %q", part)
		}

		current = child
	}

	return current, nil
}

func renderHelp(cmd *Command, out io.Writer) {
	title := cmd.Description
	if title == "" {
		title = fmt.Sprintf("%s - %s", cmd.Path, cmd.Summary)
	}

	fmt.Fprintln(out, title)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintf(out, "  %s\n", cmd.Usage)

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Global options:")
	fmt.Fprintln(out, "  --json              Output machine-readable JSON")
	fmt.Fprintln(out, "  --brief             Use compact output")
	fmt.Fprintln(out, "  --verbose           Prefer fuller human-readable output")
	fmt.Fprintln(out, "  --repo <path>       Operate on a specific repository")
	fmt.Fprintln(out, "  --help              Show help for a command")

	if len(cmd.Children) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Commands:")

		width := 0
		for _, child := range cmd.Children {
			if len(child.Use) > width {
				width = len(child.Use)
			}
		}

		for _, child := range cmd.Children {
			fmt.Fprintf(out, "  %-*s  %s\n", width, child.Use, child.Summary)
		}

		fmt.Fprintln(out)
		fmt.Fprintf(out, "Use \"%s help <command>\" or \"%s <command> --help\" for more detail.\n", cmd.Path, cmd.Path)
	}

	if len(cmd.Examples) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Examples:")
		for _, example := range cmd.Examples {
			fmt.Fprintf(out, "  %s\n", example)
		}
	}
}

func isHelpFlag(arg string) bool {
	return arg == "--help" || arg == "-h"
}

func inferCommandPath(root *Command, args []string) string {
	current := root

	for i := 0; i < len(args); i++ {
		part := args[i]
		if part == "help" {
			continue
		}

		child := current.child(part)
		if child == nil {
			break
		}
		current = child
	}

	return current.Path
}

func (c *Command) child(name string) *Command {
	for _, child := range c.Children {
		if child.Name == name {
			return child
		}
	}

	return nil
}

func rootCommand() *Command {
	note := &Command{
		Name:        "note",
		Path:        "aid note",
		Use:         "note",
		Usage:       "aid note <command> [options]",
		Summary:     "Manage notes",
		Description: "aid note - add and inspect repo-scoped notes",
		Examples: []string{
			`aid note add "Refresh token bug occurs after 401 retry"`,
			"aid note list",
		},
		Children: []*Command{
			command("add", "aid note add", "add <text>", "aid note add <text> [options]", "Add a note", noteAddCommand),
			command("list", "aid note list", "list", "aid note list [options]", "List recent notes", noteListCommand),
		},
	}

	task := &Command{
		Name:        "task",
		Path:        "aid task",
		Use:         "task",
		Usage:       "aid task <command> [options]",
		Summary:     "Manage tasks",
		Description: "aid task - add, inspect, and update tasks",
		Examples: []string{
			`aid task add "Fix VAT rounding on invoice lines"`,
			"aid task list",
			"aid task done task_12",
		},
		Children: []*Command{
			command("add", "aid task add", "add <text>", "aid task add <text> [options]", "Add a task", taskAddCommand),
			command("list", "aid task list", "list", "aid task list [options]", "List tasks", taskListCommand),
			command("done", "aid task done", "done <id>", "aid task done <id> [options]", "Mark a task as done", taskDoneCommand),
		},
	}

	decide := &Command{
		Name:        "decide",
		Path:        "aid decide",
		Use:         "decide",
		Usage:       "aid decide <command> [options]",
		Summary:     "Manage engineering decisions",
		Description: "aid decide - record and inspect engineering decisions",
		Examples: []string{
			`aid decide add "Store all monetary values as integer pence"`,
			"aid decide list",
		},
		Children: []*Command{
			command("add", "aid decide add", "add <text>", "aid decide add <text> [options]", "Record an engineering decision", decisionAddCommand),
			command("list", "aid decide list", "list", "aid decide list [options]", "List decisions", decisionListCommand),
		},
	}

	handoff := &Command{
		Name:        "handoff",
		Path:        "aid handoff",
		Use:         "handoff",
		Usage:       "aid handoff <command> [options]",
		Summary:     "Manage handoffs",
		Description: "aid handoff - generate and inspect saved handoffs",
		Examples: []string{
			"aid handoff generate",
			"aid handoff list",
		},
		Children: []*Command{
			command("generate", "aid handoff generate", "generate", "aid handoff generate [options]", "Create a structured handoff summary", handoffGenerateCommand),
			command("list", "aid handoff list", "list", "aid handoff list [options]", "List saved handoffs", handoffListCommand),
		},
	}

	history := &Command{
		Name:        "history",
		Path:        "aid history",
		Use:         "history",
		Usage:       "aid history <command> [options]",
		Summary:     "Search indexed git history",
		Description: "aid history - index and search commit history",
		Examples: []string{
			"aid history index",
			`aid history search "invoice VAT reconciliation"`,
		},
		Children: []*Command{
			command("index", "aid history index", "index", "aid history index [options]", "Index git history for search", historyIndexCommand),
			command("search", "aid history search", "search <query>", "aid history search <query> [options]", "Search indexed commit history", historySearchCommand),
		},
	}

	return &Command{
		Name:        "aid",
		Path:        "aid",
		Use:         "aid",
		Usage:       "aid <command> [options]",
		Summary:     "Local memory for coding agents and repos",
		Description: "aid - local memory for coding agents and repos",
		Examples: []string{
			"aid resume",
			`aid note add "Refresh token bug occurs after 401 retry"`,
			`aid recall "Why do we store money as integer pence?"`,
			`aid history search "invoice VAT reconciliation"`,
		},
		Children: []*Command{
			command("init", "aid init", "init", "aid init [options]", "Initialise aid in the current repository", initCommand),
			command("status", "aid status", "status", "aid status [options]", "Show repo memory status", statusCommand),
			command("resume", "aid resume", "resume", "aid resume [options]", "Show a compact working summary", resumeCommand),
			command("recall", "aid recall", "recall <query>", "aid recall <query> [options]", "Search notes, decisions, handoffs, and commits", recallCommand),
			note,
			task,
			decide,
			handoff,
			history,
		},
	}
}

func parseGlobalOptions(args []string) (output.Options, []string, error) {
	opts := output.Options{Format: output.FormatHuman}
	filtered := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			if opts.IsBrief() || opts.IsVerbose() {
				return opts, filtered, fmt.Errorf("cannot combine --json with --brief or --verbose")
			}
			opts.Format = output.FormatJSON
		case arg == "--brief":
			if opts.IsJSON() || opts.IsVerbose() {
				return opts, filtered, fmt.Errorf("cannot combine --brief with --json or --verbose")
			}
			opts.Format = output.FormatBrief
		case arg == "--verbose":
			if opts.IsJSON() || opts.IsBrief() {
				return opts, filtered, fmt.Errorf("cannot combine --verbose with --json or --brief")
			}
			opts.Format = output.FormatVerbose
		case arg == "--repo":
			if i+1 >= len(args) {
				return opts, filtered, fmt.Errorf("missing value for --repo")
			}
			opts.RepoPath = args[i+1]
			i++
		case len(arg) > len("--repo=") && arg[:7] == "--repo=":
			opts.RepoPath = arg[7:]
		default:
			filtered = append(filtered, arg)
		}
	}

	return opts, filtered, nil
}

func stubCommand(name, path, use, usage, summary string) *Command {
	return command(name, path, use, usage, summary, func(_ []string, streams Streams) error {
		fmt.Fprintf(streams.Out, "%s is scaffolded but not implemented yet.\n", path)
		return nil
	})
}

func command(name, path, use, usage, summary string, run func(args []string, streams Streams) error) *Command {
	return &Command{
		Name:        name,
		Path:        path,
		Use:         use,
		Usage:       usage,
		Summary:     summary,
		Description: fmt.Sprintf("%s - %s", path, summary),
		Run:         run,
	}
}
