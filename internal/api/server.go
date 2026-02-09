package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/sagarneeli/dist-mapreduce/internal/coordinator"
)

type Server struct {
	coordinator *coordinator.Coordinator
}

func NewServer(c *coordinator.Coordinator) *Server {
	return &Server{coordinator: c}
}

func (s *Server) Start(port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/jobs", s.handleJobs)
	mux.HandleFunc("/jobs/", s.handleJobStatus)
	mux.HandleFunc("/health", s.handleHealth)

	// We need to run this on a different port than RPC (which is on 1234)
	// Let's use 8080 for REST API
	fmt.Printf("Starting REST API on port %s\n", port)
	return http.ListenAndServe(":"+port, mux)
}

type SubmitJobRequest struct {
	Files   []string `json:"files"`
	NReduce int      `json:"nReduce"`
}

type SubmitJobResponse struct {
	JobID int `json:"id"`
}

type JobStatusResponse struct {
	ID         int    `json:"id"`
	Status     string `json:"status"`
	Files      int    `json:"files_count"`
	MapDone    int    `json:"map_tasks_completed"`
	ReduceDone int    `json:"reduce_tasks_completed"`
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubmitJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Files) == 0 || req.NReduce <= 0 {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	jobID := s.coordinator.SubmitJob(req.Files, req.NReduce)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(SubmitJobResponse{JobID: jobID}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from /jobs/{id}
	// A robust router like Chi/Gorilla would be better, but we used stdlib
	idStr := r.URL.Path[len("/jobs/"):]
	if idStr == "" {
		http.Error(w, "Missing Job ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Job ID", http.StatusBadRequest)
		return
	}

	job, ok := s.coordinator.GetJobStatus(id)
	if !ok {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Calculate progress
	mapDone := 0
	for _, t := range job.MapTasks {
		if t.Status == 2 { // TaskStatusCompleted
			mapDone++
		}
	}
	reduceDone := 0
	for _, t := range job.ReduceTasks {
		if t.Status == 2 { // TaskStatusCompleted
			reduceDone++
		}
	}

	resp := JobStatusResponse{
		ID:         job.ID,
		Status:     job.Status,
		Files:      len(job.Files),
		MapDone:    mapDone,
		ReduceDone: reduceDone,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	// w.Write value check
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
