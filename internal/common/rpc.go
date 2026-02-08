package common

import "time"

// TaskType represents the type of task (Map or Reduce).
type TaskType int

const (
	TaskTypeMap TaskType = iota
	TaskTypeReduce
)

// TaskStatus represents the status of a task.
type TaskStatus int

const (
	TaskStatusIdle TaskStatus = iota
	TaskStatusInProgress
	TaskStatusCompleted
	TaskStatusFailed
)

// TaskArgs holds the arguments for a task request.
type TaskArgs struct {
	WorkerID string
}

// TaskReply holds the task details assigned to a worker.
type TaskReply struct {
	JobID     int
	TaskType  TaskType
	TaskID    int
	FileName  string // For Map tasks
	NReduce   int    // Number of reduce tasks
	NMap      int    // Number of map tasks
	Timestamp time.Time
	Task      *Task
}

// Task represents a unit of work.
type Task struct {
	ID        int
	JobID     int
	Type      TaskType
	Status    TaskStatus
	FileName  string
	StartTime time.Time
	WorkerID  string
}

// ReportTaskArgs holds arguments for reporting task completion.
type ReportTaskArgs struct {
	JobID    int
	TaskID   int
	TaskType TaskType
	WorkerID string
}

// ReportTaskReply holds the response for task completion report.
type ReportTaskReply struct {
	Ack bool
}
