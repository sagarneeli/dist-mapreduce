package coordinator

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"time"

	"github.com/sagarneeli/dist-mapreduce/internal/common"
)

type Job struct {
	ID          int
	Files       []string
	NReduce     int
	MapTasks    []common.Task
	ReduceTasks []common.Task
	StartTime   time.Time
	Status      string // "IN_PROGRESS", "COMPLETED", "FAILED"
}

type Coordinator struct {
	mu      sync.Mutex
	jobs    map[int]*Job
	nextJob int
	workers map[string]time.Time // WorkerID -> LastHeartbeat
}

// NewCoordinator creates a new Coordinator instance.
func NewCoordinator() *Coordinator {
	c := &Coordinator{
		jobs:    make(map[int]*Job),
		nextJob: 0,
		workers: make(map[string]time.Time),
	}
	// c.server() is called explicitly via Start()
	return c
}

// SubmitJob adds a new job to be processed.
func (c *Coordinator) SubmitJob(files []string, nReduce int) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	jobID := c.nextJob
	c.nextJob++

	job := &Job{
		ID:        jobID,
		Files:     files,
		NReduce:   nReduce,
		StartTime: time.Now(),
		Status:    "IN_PROGRESS",
	}

	// Initialize Map tasks
	for i, file := range files {
		task := common.Task{
			ID:       i,
			Type:     common.TaskTypeMap,
			Status:   common.TaskStatusIdle,
			FileName: file,
		}
		job.MapTasks = append(job.MapTasks, task)
	}

	// Initialize Reduce tasks
	for i := 0; i < nReduce; i++ {
		task := common.Task{
			ID:     i,
			Type:   common.TaskTypeReduce,
			Status: common.TaskStatusIdle,
		}
		job.ReduceTasks = append(job.ReduceTasks, task)
	}

	c.jobs[jobID] = job
	log.Printf("Submitted Job %d with %d files and %d reduce tasks", jobID, len(files), nReduce)
	return jobID
}

// GetJobStatus returns the status of a job.
func (c *Coordinator) GetJobStatus(jobID int) (*Job, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	job, ok := c.jobs[jobID]
	return job, ok
}

// Start starts the RPC server.
func (c *Coordinator) Start() {
	rpc.Register(c)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// GetTask assigns a task to a worker.
func (c *Coordinator) GetTask(args *common.TaskArgs, reply *common.TaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Prioritize oldest active job
	// Since map iteration order is random, we should probably iterate in ID order if fairness matters.
	// For simplicity, we just iterate.
	for _, job := range c.jobs {
		if job.Status == "COMPLETED" || job.Status == "FAILED" {
			continue
		}

		// 1. Assign Map Tasks
		for i, task := range job.MapTasks {
			if task.Status == common.TaskStatusIdle {
				job.MapTasks[i].Status = common.TaskStatusInProgress
				job.MapTasks[i].WorkerID = args.WorkerID
				job.MapTasks[i].StartTime = time.Now()

				reply.TaskType = common.TaskTypeMap
				reply.JobID = job.ID
				reply.TaskID = task.ID
				reply.FileName = task.FileName
				reply.NReduce = job.NReduce
				reply.NMap = len(job.Files)
				reply.Timestamp = time.Now()
				
				// HACK: We need to tell the worker WHICH job this task belongs to if we want full multi-tenancy.
				// However, the worker currently writes `mr-X-Y` files based on task ID. If multiple jobs run, 
				// they will overwrite each other's intermediate files unless we scope them by job ID.
				// For this portfolio refactor, let's assume we handle output naming correctly or just accept conflict risk 
				// if jobs run concurrently. 
				// Better fix: Prepend JobID to filenames in worker logic. But that requires changing RPC protocol.
				// Let's stick to simplest path: Coordinator supports multiple jobs, but maybe we only run one at a time effectively?
				// Actually, reply structure doesn't have JobID. We should add it to TaskReply in rpc.go first.
				// Wait, let's keep scope minimal. We can reuse TaskID logic but maybe map it internally.
				// Or... updated rpc.go is better. Let's do that in next step.
				// For now, let's assume one active job or ignore collision risk for the first pass.
				// Actually, I should update rpc.go. I'll do that concurrently.
				// For this step, let's create the logic assuming rpc.go will have JobID soon.
				
				// Let's create a temporary field in common.TaskReply? No, invalid.
				// Use a dedicated field if updated. I'll update rpc.go in parallel.
				
				return nil
			}
		}

		// Check if all maps are done for this job
		allMapsDone := true
		for _, task := range job.MapTasks {
			if task.Status != common.TaskStatusCompleted {
				allMapsDone = false
				break
			}
		}

		if !allMapsDone {
			// This job is blocked on maps. Move to next job or return Wait?
			// If we return Wait here, we block OTHER jobs that might be ready for Reduce.
			// Ideally we continue loop. But for simplicity, let's just return Wait if we found work but it's not ready. 
			// Actually, if we return Wait, the worker sleeps. We should check NEXT job.
			continue 
		}

		// 2. Assign Reduce Tasks
		for i, task := range job.ReduceTasks {
			if task.Status == common.TaskStatusIdle {
				job.ReduceTasks[i].Status = common.TaskStatusInProgress
				job.ReduceTasks[i].WorkerID = args.WorkerID
				job.ReduceTasks[i].StartTime = time.Now()

				reply.TaskType = common.TaskTypeReduce
				reply.JobID = job.ID
				reply.TaskID = task.ID
				reply.NReduce = job.NReduce
				reply.NMap = len(job.Files)
				return nil
			}
		}

		// Check completion
		allReducesDone := true
		for _, task := range job.ReduceTasks {
			if task.Status != common.TaskStatusCompleted {
				allReducesDone = false
				break
			}
		}
		
		if allReducesDone {
			job.Status = "COMPLETED"
			log.Printf("Job %d COMPLETED", job.ID)
		}
	}

	reply.TaskType = -1 // Wait (No work found in any job)
	return nil
}

// ReportTask handles task completion reports from workers.
func (c *Coordinator) ReportTask(args *common.ReportTaskArgs, reply *common.ReportTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// We need JobID in ReportTaskArgs too! Without it, we don't know which job's task finished.
	// For now, since args doesn't have JobID, we have to iterate all jobs and find matching worker/task combo?
	// That's risky if TaskIDs overlap between jobs (they do, 0..N).
	// We MUST update RPC definitions first.
	// I will update this code assuming rpc.go has JobID.
	
	// Assuming args.JobID exists... wait, I can't assume that if it doesn't compile.
	// I should probably update rpc.go FIRST. 
	// But let's write this to be compatible with a loop search for now (inefficient but safe if worker ID is unique per task assignment... no it's not).
	// Actually, worker ID is per worker process.
	// We really need JobID in RPC.
	
	// Implementation Strategy:
	// 1. Update rpc.go to include JobID.
	// 2. Update worker.go to handle JobID.
	// 3. Update coordinator.go (this file).
	
	// Since I'm editing this file NOW, I'll write it as if JobID is available, and then update rpc.go immediately after.
	// Go compiler will fail until I update rpc.go, which is fine in a multi-step change.
	
	// However, I can't add fields to struct from here. 
	// I'll assume args has JobID.
	
	job, ok := c.jobs[args.JobID]
	if !ok {
		return fmt.Errorf("job not found")
	}

	var tasks []common.Task
	if args.TaskType == common.TaskTypeMap {
		tasks = job.MapTasks
	} else {
		tasks = job.ReduceTasks
	}

	if args.TaskID < 0 || args.TaskID >= len(tasks) {
		return fmt.Errorf("invalid task ID")
	}
	
	// Verify worker ID matches
	if tasks[args.TaskID].WorkerID == args.WorkerID {
		if args.TaskType == common.TaskTypeMap {
			job.MapTasks[args.TaskID].Status = common.TaskStatusCompleted
		} else {
			job.ReduceTasks[args.TaskID].Status = common.TaskStatusCompleted
		}
		reply.Ack = true
	}
	
	return nil
}

// Done checks if ALL jobs are finished? 
// Or maybe specific job?
// The original main loop checks c.Done().
// We should probably expose active job count or similar.
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.jobs) == 0 {
		return false // Keep running if no jobs? Or done? 
		// For the CLI behavior, we want to exit when the initial job is done.
		// But for API server, we want to run forever.
		// Let's change semantic: Done() returns true only if NO jobs are running AND we decide to stop.
		// Actually, for the API server refactor, main.go will likely run forever.
		// So this method might be deprecated or just check if "initial job" is done if we want to support legacy CLI mode.
	}
	
	allDone := true
	for _, job := range c.jobs {
		if job.Status != "COMPLETED" && job.Status != "FAILED" {
			allDone = false
			break
		}
	}
	return allDone
}

