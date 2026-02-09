package worker

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sagarneeli/dist-mapreduce/internal/common"
)

// KeyValue is a type for map tasks
type KeyValue struct {
	Key   string
	Value string
}

// MapFunc is the signature for the map function
func MapFunc(filename string, contents string) []KeyValue {
	// Simple word count map function
	// Split by non-alphanumeric characters
	tokens := strings.FieldsFunc(contents, func(r rune) bool {
		return !('a' <= r && r <= 'z' || 'A' <= r && r <= 'Z')
	})

	kva := []KeyValue{}
	for _, w := range tokens {
		if len(w) > 0 {
			kva = append(kva, KeyValue{Key: w, Value: "1"})
		}
	}
	return kva
}

// ReduceFunc is the signature for the reduce function
func ReduceFunc(key string, values []string) string {
	// Simple word count reduce function
	return fmt.Sprintf("%d", len(values))
}

func Worker(coordinatorHost string) {
	workerID := fmt.Sprintf("worker-%d", os.Getpid())
	log.Printf("Worker %s started", workerID)

	for {
		args := common.TaskArgs{WorkerID: workerID}
		reply := common.TaskReply{}

		if !call(coordinatorHost, "Coordinator.GetTask", &args, &reply) {
			log.Println("Coordinator unreachable, exiting.")
			return
		}

		switch reply.TaskType {
		case common.TaskTypeMap:
			doMap(reply.JobID, reply.TaskID, reply.FileName, reply.NReduce, MapFunc)
			report(coordinatorHost, reply.JobID, reply.TaskID, common.TaskTypeMap, workerID)
		case common.TaskTypeReduce:
			doReduce(reply.JobID, reply.TaskID, reply.NMap, ReduceFunc)
			report(coordinatorHost, reply.JobID, reply.TaskID, common.TaskTypeReduce, workerID)
		case -1: // Wait
			time.Sleep(time.Second)
		case -2: // Done
			log.Println("No tasks available, waiting...")
			time.Sleep(time.Second)
		}
	}
}

func doMap(jobID int, taskID int, filename string, nReduce int, mapF func(string, string) []KeyValue) {
	log.Printf("Starting Map Task %d for Job %d file %s", taskID, jobID, filename)
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	kva := mapF(filename, string(content))

	// Partitioning
	buckets := make([][]KeyValue, nReduce)
	for _, kv := range kva {
		bucket := ihash(kv.Key) % nReduce
		buckets[bucket] = append(buckets[bucket], kv)
	}

	for i := 0; i < nReduce; i++ {
		// Include JobID in filename to prevent collisions
		oname := fmt.Sprintf("mr-%d-%d-%d", jobID, taskID, i)
		file, _ := os.Create(oname)
		enc := json.NewEncoder(file)
		for _, kv := range buckets[i] {
			if err := enc.Encode(&kv); err != nil {
				log.Fatalf("cannot encode %v: %v", kv, err)
			}
		}
		file.Close()
	}
	log.Printf("Finished Map Task %d Job %d", taskID, jobID)
}

func doReduce(jobID int, taskID int, nMap int, reduceF func(string, []string) string) {
	log.Printf("Starting Reduce Task %d for Job %d", taskID, jobID)
	intermediate := make(map[string][]string)

	for i := 0; i < nMap; i++ {
		// Read from JobID namespaced files
		iname := fmt.Sprintf("mr-%d-%d-%d", jobID, i, taskID)
		file, err := os.Open(iname)
		if err != nil {
			log.Printf("Failed to open intermediate file %s: %v", iname, err)
			continue
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate[kv.Key] = append(intermediate[kv.Key], kv.Value)
		}
		file.Close()
	}

	keys := []string{}
	for k := range intermediate {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	oname := fmt.Sprintf("mr-out-%d-%d", jobID, taskID)
	ofile, _ := os.Create(oname)

	for _, k := range keys {
		output := reduceF(k, intermediate[k])
		fmt.Fprintf(ofile, "%v %v\n", k, output)
	}
	ofile.Close()
	log.Printf("Finished Reduce Task %d Job %d", taskID, jobID)
}

func report(coordinatorHost string, jobID int, taskID int, taskType common.TaskType, workerID string) {
	args := common.ReportTaskArgs{JobID: jobID, TaskID: taskID, TaskType: taskType, WorkerID: workerID}
	reply := common.ReportTaskReply{}
	call(coordinatorHost, "Coordinator.ReportTask", &args, &reply)
}

func call(host, rpcname string, args interface{}, reply interface{}) bool {
	c, err := rpc.DialHTTP("tcp", host+":1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}

func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}
