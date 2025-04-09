// queue.go
package utils

import (
	"context"
	"database/sql"
	"log"
	"time"

	"AllinB/src/consts"
)

// DB는 데이터베이스 연결을 저장합니다.
var DB *sql.DB

// EnqueueJobFunc는 작업을 큐에 추가하는 함수의 타입입니다.
type EnqueueJobFunc func(job Job)

// Job 구조체는 비동기 작업을 표현합니다.
type Job struct {
	Name     string
	Data     map[string]interface{}
	Priority int // 높을수록 우선순위 높음
}

// 작업 큐에 추가하기 위한 함수 참조
var EnqueueJobHandler EnqueueJobFunc

// SetEnqueueJobFunc는 작업을 큐에 추가하는 함수를 설정합니다.
func SetEnqueueJobFunc(fn EnqueueJobFunc) {
	EnqueueJobHandler = fn
}

// jobQueue는 버퍼링된 채널로, 최대 100개의 작업을 저장할 수 있습니다.
var jobQueue = make(chan Job, 100)

// EnqueueJob은 작업을 큐에 추가합니다.
func EnqueueJob(job Job) {
	select {
	case jobQueue <- job:
		log.Printf("Job enqueued: %s", job.Name)
	default:
		log.Printf("Job queue full, dropping job: %s", job.Name)
	}
}

// StartJobWorker는 백그라운드에서 큐의 작업을 처리하는 워커를 시작합니다.
func StartJobWorker() {
	go func() {
		for job := range jobQueue {
			processJob(job)
		}
	}()
}

// 워커 수를 구성 가능하게 만듦
func StartJobWorkers(workerCount int) {
	for i := 0; i < workerCount; i++ {
		go func(id int) {
			log.Printf("Worker %d started", id)
			for job := range jobQueue {
				processJob(job)
			}
		}(i)
	}
}

// processJob은 작업을 처리합니다. (여기서는 단순 로그 출력과 1초 Sleep으로 시뮬레이션)
func processJob(job Job) {
	timeout := time.Duration(consts.DEFAULT_WORK_TIMEOUT) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan bool)
	go func() {
		// 실제 작업 처리
		log.Printf("Processing job: %s", job.Name)
		// ... 작업 로직
		done <- true
	}()

	select {
	case <-done:
		log.Printf("Job processed: %s", job.Name)
	case <-ctx.Done():
		log.Printf("Job timed out: %s", job.Name)
	}
}
