package todo

import (
	"strings"

	"github.com/google/uuid"
	"github.com/miruken-go/miruken/es/aggergate"
	"github.com/miruken-go/miruken/es/events"
	"github.com/miruken-go/miruken/handles"
)

type (
	// List is a "todo" list.
	List struct {
		Id      uuid.UUID
		Version int

		tasks   []string
		archive []string
	}

	AddTask struct {
		Task string
	}

	TaskAdded struct {
		Task string
	}

	RemoveTask struct {
		Task string
	}

	TaskRemoved struct {
		Task string
	}

	CompleteTasks struct {
		Tasks []string
	}

	TasksCompleted struct {
		Tasks []string
	}
)


func (l *List) Constructor(
	_ *struct{
		aggergate.Root  `agg:"name=todo.list"`
	},
) {
}

func (l *List) AddTask(
	_ *handles.It, add AddTask,
) events.Stream {
	task := add.Task
	if l.Contains(task) {
		return nil
	}
	return events.Append(TaskAdded{task})
}

func (l *List) RemoveTask(
	_ *handles.It, remove RemoveTask,
) events.Stream {
	task := remove.Task
	if !l.Contains(task) {
		return nil
	}
	return events.Append(TaskRemoved{task})
}

func (l *List) CompleteTasks(
	_ *handles.It, complete CompleteTasks,
) events.Stream {
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

	return events.Append(TasksCompleted{done})
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

func (l *List) ApplyTaskAdded(
	added TaskAdded,
) {
	l.tasks = append(l.tasks, added.Task)
}

func (l *List) ApplyTaskRemoved(
	removed TaskRemoved,
) {
	lt := strings.ToLower(removed.Task)
	for i, task := range l.tasks {
		if strings.ToLower(task) == lt {
			l.tasks = append(l.tasks[:i], l.tasks[i+1:]...)
			return
		}
	}
}

func (l *List) ApplyTasksCompleted(
	completed TasksCompleted,
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