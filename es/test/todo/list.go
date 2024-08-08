package todo

import (
	"strings"

	"github.com/google/uuid"
	"github.com/miruken-go/miruken/es/aggergate"
	"github.com/miruken-go/miruken/es/command"
	"github.com/miruken-go/miruken/es/event"
)

type (
	// List is a "todo" list.
	List struct {
		Id      uuid.UUID
		Version int

		tasks   []string
		archive []string
	}
)

// commands

type (
	AddTask struct {
		Task string
	}

	RemoveTask struct {
		Task string
	}

	CompleteTasks struct {
		Tasks []string
	}
)

// events

type (
	TaskAdded struct {
		Task string
	}

	TaskRemoved struct {
		Task string
	}

	TasksCompleted struct {
		Tasks []string
	}
)


// List

func (l *List) Constructor(
	_ *aggergate.Root,
) {
}

// commands

func (l *List) AddTask(
	_ *command.Handler, add AddTask,
) event.Stream {
	task := add.Task
	if l.Contains(task) {
		return nil
	}
	return event.Append(TaskAdded{task})
}

func (l *List) RemoveTask(
	_ *command.Handler, remove RemoveTask,
) event.Stream {
	task := remove.Task
	if !l.Contains(task) {
		return nil
	}
	return event.Append(TaskRemoved{task})
}

func (l *List) CompleteTasks(
	_ *struct {
		command.Handler `command:"name=completeTasks"`
	}, complete CompleteTasks,
) event.Stream {
	tasks := complete.Tasks
	if len(tasks) == 0 {
		return nil
	}

	var done []string
	for _, task := range tasks {
		lt := strings.ToLower(task)
		for _, t := range l.tasks {
			if strings.ToLower(t) == lt {
				done = append(done, task)
				break
			}
		}
	}

	if len(done) == 0 {
		return nil
	}

	return event.Append(TasksCompleted{done})
}

func (l *List) Contains(task string) bool {
	task = strings.ToLower(task)
	for _, t := range l.tasks {
		if strings.ToLower(t) == task {
			return true
		}
	}
	return false
}

// events

func (l *List) TaskAdded(
	_ *event.Handler, added TaskAdded,
) {
	l.tasks = append(l.tasks, added.Task)
}

func (l *List) TaskRemoved(
	_ *event.Handler, removed TaskRemoved,
) {
	lt := strings.ToLower(removed.Task)
	for i, task := range l.tasks {
		if strings.ToLower(task) == lt {
			l.tasks = append(l.tasks[:i], l.tasks[i+1:]...)
			return
		}
	}
}

func (l *List) TasksCompleted(
	_ *event.Handler, completed TasksCompleted,
) {
	for _, task := range completed.Tasks {
		lt := strings.ToLower(task)
		for i, t := range l.tasks {
			if strings.ToLower(t) == lt {
				l.archive = append(l.archive, task)
				l.tasks = append(l.tasks[:i], l.tasks[i+1:]...)
				break
			}
		}
	}
}
