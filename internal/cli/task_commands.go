package cli

import (
	"context"
	"fmt"

	"github.com/forjd/aid/internal/output"
	"github.com/forjd/aid/internal/store"
)

func taskAddCommand(args []string, streams Streams) error {
	text, err := joinArgs(args, "task text")
	if err != nil {
		return err
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	task, err := runtime.store.AddTask(context.Background(), store.AddTaskInput{
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
		return fmt.Errorf("task list does not accept arguments")
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	tasks, err := runtime.store.ListTasks(context.Background(), runtime.repo.ID, defaultListLimit)
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
		return fmt.Errorf("task %s expects exactly one task id", taskCommandName(status))
	}

	taskID, err := store.ParseTaskRef(args[0])
	if err != nil {
		return err
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	task, err := runtime.store.UpdateTaskStatus(context.Background(), runtime.repo.ID, taskID, status)
	if err != nil {
		return err
	}

	return output.RenderTaskStatusUpdated(streams.Out, streams.Options, taskCommandName(status), taskStatusVerb(status), task)
}
