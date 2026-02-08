package coordinator

import (
	"testing"

	"github.com/sagarneeli/dist-mapreduce/internal/common"
)

func TestCoordinatorSequence(t *testing.T) {
	files := []string{"file1.txt", "file2.txt"}
	nReduce := 2
	c := NewCoordinator()
	jobID := c.SubmitJob(files, nReduce)

	if c.Done() {
		t.Error("Coordinator should not be done upon initialization")
	}

	// 1. Get Task (Should be Map)
	args := &common.TaskArgs{WorkerID: "w1"}
	reply := &common.TaskReply{}
	err := c.GetTask(args, reply)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if reply.TaskType != common.TaskTypeMap {
		t.Errorf("Expected Map task, got %v", reply.TaskType)
	}
	if reply.JobID != jobID {
		t.Errorf("Expected JobID %d, got %d", jobID, reply.JobID)
	}

	// 2. Report Map Task Done
	reportArgs := &common.ReportTaskArgs{
		JobID:    jobID,
		TaskID:   reply.TaskID,
		TaskType: common.TaskTypeMap,
		WorkerID: "w1",
	}
	reportReply := &common.ReportTaskReply{}
	err = c.ReportTask(reportArgs, reportReply)
	if err != nil {
		t.Fatalf("ReportTask failed: %v", err)
	}
	if !reportReply.Ack {
		t.Error("ReportTask not acknowledged")
	}

	// 3. Get another task (Should be the second Map)
	reply2 := &common.TaskReply{}
	c.GetTask(args, reply2)
	if reply2.TaskType != common.TaskTypeMap {
		t.Errorf("Expected second Map task, got %v", reply2.TaskType)
	}

	// 4. Report second Map Task Done
	reportArgs.TaskID = reply2.TaskID
	err = c.ReportTask(reportArgs, reportReply)
	if err != nil {
		t.Fatalf("ReportTask failed: %v", err)
	}

	// 5. Get task (Should be Reduce, since all Maps are done)
	reply3 := &common.TaskReply{}
	c.GetTask(args, reply3)
	if reply3.TaskType != common.TaskTypeReduce {
		t.Errorf("Expected Reduce task, got %v", reply3.TaskType)
	}
}

func TestCoordinator_Done(t *testing.T) {
	files := []string{"f1"}
	c := NewCoordinator()
	jobID := c.SubmitJob(files, 1)

	// Finish map
	job, _ := c.GetJobStatus(jobID)
	job.MapTasks[0].Status = common.TaskStatusCompleted

	// Finish reduce
	job.ReduceTasks[0].Status = common.TaskStatusCompleted
	job.Status = "COMPLETED" // Logic in GetTask updates this, but manually setting for unit test shortcut

	if !c.Done() {
		t.Error("Coordinator should be done when all tasks are completed")
	}
}

func TestCoordinator_Wait(t *testing.T) {
	files := []string{"f1"}
	c := NewCoordinator()
	jobID := c.SubmitJob(files, 1)

	// Assign the only map task
	job, _ := c.GetJobStatus(jobID)
	job.MapTasks[0].Status = common.TaskStatusInProgress

	// Request another task -> Should be Wait (-1) because map not done yet
	args := &common.TaskArgs{WorkerID: "w2"}
	reply := &common.TaskReply{}
	c.GetTask(args, reply)

	if reply.TaskType != -1 {
		t.Errorf("Expected Wait (-1), got %v", reply.TaskType)
	}
}
