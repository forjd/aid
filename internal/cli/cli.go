package cli

import (
	"fmt"
	"io"
)

type Streams struct {
	Out io.Writer
	Err io.Writer
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
	streams := Streams{
		Out: stdout,
		Err: stderr,
	}

	if err := dispatch(rootCommand(), args, streams); err != nil {
		fmt.Fprintf(streams.Err, "Error: %v\n", err)
		return 1
	}

	return 0
}

func dispatch(cmd *Command, args []string, streams Streams) error {
	if len(args) == 0 || isHelpFlag(args[0]) {
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
		Usage:       "aid note <command>",
		Summary:     "Manage notes",
		Description: "aid note - add and inspect repo-scoped notes",
		Examples: []string{
			`aid note add "Refresh token bug occurs after 401 retry"`,
			"aid note list",
		},
		Children: []*Command{
			stubCommand("add", "aid note add", "add <text>", "aid note add <text>", "Add a note"),
			stubCommand("list", "aid note list", "list", "aid note list", "List recent notes"),
		},
	}

	task := &Command{
		Name:        "task",
		Path:        "aid task",
		Use:         "task",
		Usage:       "aid task <command>",
		Summary:     "Manage tasks",
		Description: "aid task - add, inspect, and update tasks",
		Examples: []string{
			`aid task add "Fix VAT rounding on invoice lines"`,
			"aid task list",
			"aid task done task_12",
		},
		Children: []*Command{
			stubCommand("add", "aid task add", "add <text>", "aid task add <text>", "Add a task"),
			stubCommand("list", "aid task list", "list", "aid task list", "List tasks"),
			stubCommand("done", "aid task done", "done <id>", "aid task done <id>", "Mark a task as done"),
		},
	}

	decide := &Command{
		Name:        "decide",
		Path:        "aid decide",
		Use:         "decide",
		Usage:       "aid decide <command>",
		Summary:     "Manage engineering decisions",
		Description: "aid decide - record and inspect engineering decisions",
		Examples: []string{
			`aid decide add "Store all monetary values as integer pence"`,
			"aid decide list",
		},
		Children: []*Command{
			stubCommand("add", "aid decide add", "add <text>", "aid decide add <text>", "Record an engineering decision"),
			stubCommand("list", "aid decide list", "list", "aid decide list", "List decisions"),
		},
	}

	handoff := &Command{
		Name:        "handoff",
		Path:        "aid handoff",
		Use:         "handoff",
		Usage:       "aid handoff <command>",
		Summary:     "Manage handoffs",
		Description: "aid handoff - generate and inspect saved handoffs",
		Examples: []string{
			"aid handoff generate",
			"aid handoff list",
		},
		Children: []*Command{
			stubCommand("generate", "aid handoff generate", "generate", "aid handoff generate", "Create a structured handoff summary"),
			stubCommand("list", "aid handoff list", "list", "aid handoff list", "List saved handoffs"),
		},
	}

	history := &Command{
		Name:        "history",
		Path:        "aid history",
		Use:         "history",
		Usage:       "aid history <command>",
		Summary:     "Search indexed git history",
		Description: "aid history - index and search commit history",
		Examples: []string{
			"aid history index",
			`aid history search "invoice VAT reconciliation"`,
		},
		Children: []*Command{
			stubCommand("index", "aid history index", "index", "aid history index", "Index git history for search"),
			stubCommand("search", "aid history search", "search <query>", "aid history search <query>", "Search indexed commit history"),
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
			stubCommand("init", "aid init", "init", "aid init", "Initialise aid in the current repository"),
			stubCommand("status", "aid status", "status", "aid status", "Show repo memory status"),
			stubCommand("resume", "aid resume", "resume", "aid resume", "Show a compact working summary"),
			stubCommand("recall", "aid recall", "recall <query>", "aid recall <query>", "Search notes, decisions, handoffs, and commits"),
			note,
			task,
			decide,
			handoff,
			history,
		},
	}
}

func stubCommand(name, path, use, usage, summary string) *Command {
	return &Command{
		Name:        name,
		Path:        path,
		Use:         use,
		Usage:       usage,
		Summary:     summary,
		Description: fmt.Sprintf("%s - %s", path, summary),
		Run: func(_ []string, streams Streams) error {
			fmt.Fprintf(streams.Out, "%s is scaffolded but not implemented yet.\n", path)
			return nil
		},
	}
}
