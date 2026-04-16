package cli

import (
	"strings"

	"github.com/forjd/aid/internal/output"
	"github.com/forjd/aid/internal/store"
)

func taskAddCommand(args []string, streams Streams) error {
	text, err := joinArgs(args, "task text")
	if err != nil {
		return err
	}

	ctx := streams.context()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	task, err := runtime.store.AddTask(ctx, store.AddTaskInput{
		RepoID: runtime.repo.ID,
		Branch: runtime.env.Branch,
		Scope:  store.ScopeBranch,
		Text:   text,
		Status: store.TaskOpen,
	})
	if err != nil {
		return err
	}

	return output.RenderTaskAdded(streams.Out, streams.Options, task)
}

func taskListCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return newError(ErrCodeUsage, "task list does not accept arguments")
	}

	ctx := streams.context()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	tasks, err := runtime.store.ListTasks(ctx, runtime.repo.ID, runtime.env.Branch, defaultListLimit)
	if err != nil {
		return err
	}

	return output.RenderTasks(streams.Out, streams.Options, tasks)
}

func taskDoneCommand(args []string, streams Streams) error {
	return taskStatusCommand(args, streams, store.TaskDone)
}

func taskStartCommand(args []string, streams Streams) error {
	return taskStatusCommand(args, streams, store.TaskInProgress)
}

func taskBlockCommand(args []string, streams Streams) error {
	return taskStatusCommand(args, streams, store.TaskBlocked)
}

func taskReopenCommand(args []string, streams Streams) error {
	return taskStatusCommand(args, streams, store.TaskOpen)
}

func taskStatusCommand(args []string, streams Streams, status store.TaskStatus) error {
	if len(args) != 1 {
		return newError(ErrCodeUsage, "task %s expects exactly one task id", taskCommandName(status))
	}

	taskID, err := store.ParseTaskRef(args[0])
	if err != nil {
		return newError(ErrCodeInvalidInput, "%s", err.Error())
	}

	ctx := streams.context()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	task, err := runtime.store.UpdateTaskStatus(ctx, runtime.repo.ID, taskID, status)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return newError(ErrCodeNotFound, "%s", err.Error())
		}
		return err
	}

	return output.RenderTaskStatusUpdated(streams.Out, streams.Options, taskCommandName(status), taskStatusVerb(status), task)
}
