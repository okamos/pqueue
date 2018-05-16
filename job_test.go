package pqueue

import (
	"strings"
	"testing"
	"time"

	validator "gopkg.in/go-playground/validator.v9"
)

func TestJobSave(t *testing.T) {
	job := NewJob("test", []byte(`{"test":"job"}`), 5)
	err := job.Save()
	if err != nil {
		t.Error(err)
	}
}

func TestJobSaveNameRequired(t *testing.T) {
	job := NewJob("", nil, 5)
	err := job.Save()
	if err == nil || len(err.(validator.ValidationErrors)) == 0 {
		t.Error("Job.Name should be required")
	}
	for _, err := range err.(validator.ValidationErrors) {
		if err.Field() == "Name" && err.Tag() != "required" {
			t.Error("Job.Name should be required")
		}
	}
}

func TestJobSaveStatusShould0to2(t *testing.T) {
	job := NewJob("test", nil, 5)
	job.Status = 3
	err := job.Save()
	if err == nil || len(err.(validator.ValidationErrors)) == 0 {
		t.Error("Job.Status should be 0 to 2")
	}
	for _, err := range err.(validator.ValidationErrors) {
		if err.Field() == "Status" && err.Tag() != "lte" {
			t.Error("Job.Status should be 0 to 2")
		}
	}
}

func TestJobTimeoutGreaterThan0(t *testing.T) {
	job := NewJob("test", nil, 0)
	err := job.Save()
	if err == nil || len(err.(validator.ValidationErrors)) == 0 {
		t.Error("Job.Timeout should be greater than 0")
	}
	for _, err := range err.(validator.ValidationErrors) {
		if err.Field() == "Timeout" && err.Tag() != "gt" {
			t.Error("Job.Timeout should be greater than 0")
		}
	}
}

func TestJobPayloadShouldBeJSON(t *testing.T) {
	job := NewJob("test", []byte("invalid"), 5)
	err := job.Save()
	if !strings.Contains(err.Error(), "invalid character") {
		t.Error(err)
	}
}

func TestLockJobsLocksAndReturnsJobs(t *testing.T) {
	TruncateJob()

	for i := 0; i < 5; i++ {
		job := NewJob("test", nil, 5)
		job.Priority = i
		job.Save()
	}

	jobs, _ := LockJobs(2)
	if len(jobs) != 2 {
		t.Errorf("processing jobs length expect 2, actual %d", len(jobs))
	}

	jobs, _ = LockJobs(5)
	beforeJob := jobs[0]
	for _, job := range jobs {
		if job.Priority > beforeJob.Priority {
			t.Error("LockJobs should order by priority descending")
		}
		beforeJob = job
	}
}

func TestDeleteJob(t *testing.T) {
	TruncateJob()

	job := NewJob("test", nil, 5)
	job.RunAfter = time.Now().Add(4 * time.Hour)
	job.Save()

	jobs, _ := EnqueuedJobsByName("test")
	if len(jobs) != 1 {
		t.Errorf("enqueued jobs expect 1, actual %d", len(jobs))
		return
	}

	job.Delete()
	jobs, _ = EnqueuedJobsByName("test")
	if len(jobs) != 0 {
		t.Errorf("enqueued jobs expect 0, actual %d", len(jobs))
		return
	}
}

func TestCompleteJob(t *testing.T) {
	TruncateJob()

	job := NewJob("test", nil, 5)
	job.Save()

	job.Complete()

	jobs, _ := LockJobs(1)
	if len(jobs) != 0 {
		t.Errorf("processing jobs expect 0, actual %d", len(jobs))
		return
	}
	jobs, _ = ProcessedJobs(time.Time{}, 0)
	if len(jobs) != 1 {
		t.Errorf("processed jobs expect 1, actual %d", len(jobs))
		return
	}
	if jobs[0].ID != job.ID {
		t.Error("invalid processed job")
	}
}

func TestFailJob(t *testing.T) {
	TruncateJob()

	job := NewJob("test", nil, 5)
	job.Save()

	job.Fail("")

	jobs, _ := LockJobs(1)
	if len(jobs) != 0 {
		t.Errorf("processing jobs expect 0, actual %d", len(jobs))
	}

	job = NewJob("test", nil, 5)
	job.RunAfter = time.Now().Add(-4 * time.Hour)
	job.Save()
	job.Fail("")

	if job.Status != 0 {
		t.Error("Job.Status should be 0")
	}
	if job.RunCount != 1 {
		t.Error("Job.RunCount should be 1")
	}
	job.Fail("")

	if job.Status != 0 {
		t.Error("Job.Status should be 0")
	}
	if job.RunCount != 2 {
		t.Error("Job.RunCount should be 2")
	}
	job.Fail("")

	jobs, _ = FailedJobs(time.Time{}, 0)
	if len(jobs) != 1 {
		t.Errorf("failed jobs expect 1, actual %d", len(jobs))
	}
	if jobs[0].ID != job.ID {
		t.Error("invalid failed job")
	}
}

func TestProccssingJobs(t *testing.T) {
	TruncateJob()

	for i := 0; i <= 4; i++ {
		job := NewJob("test", nil, 5)
		job.RunAfter = time.Now().Add(-4 * time.Hour)
		job.Save()
	}
	LockJobs(2)
	jobs, _ := ProcessingJobs()
	if len(jobs) != 2 {
		t.Errorf("expect ProcessingJobs 2, actual %d", len(jobs))
	}
	LockJobs(1)
	jobs, _ = ProcessingJobs()
	if len(jobs) != 3 {
		t.Errorf("expect ProcessingJobs 3, actual %d", len(jobs))
	}
	jobs[1].Complete()
	jobs, _ = ProcessingJobs()
	if len(jobs) != 2 {
		t.Errorf("expect ProcessingJobs 2, actual %d", len(jobs))
	}
}

func TestProcessedJobs(t *testing.T) {
	TruncateJob()

	for i := 1; i <= 51; i++ {
		job := NewJob("test", nil, uint(i))
		job.RunAfter = time.Now().Add(-4 * time.Hour)
		job.Save()
		job.Complete()
	}

	jobs, _ := ProcessedJobs(time.Time{}, 0)
	if len(jobs) != 25 {
		t.Errorf("expect ProcessedJobs 25, actual %d", len(jobs))
	}
	if jobs[0].Timeout != 51 {
		t.Errorf("expect timeout 51, actual %d", jobs[0].Timeout)
	}
	jobs, _ = ProcessedJobs(jobs[24].RunAfter, jobs[24].ID)
	if len(jobs) != 25 {
		t.Errorf("expect ProcessedJobs 25, actual %d", len(jobs))
	}
	if jobs[0].Timeout != 26 {
		t.Errorf("expect timeout 26, actual %d", jobs[0].Timeout)
	}
	jobs, _ = ProcessedJobs(jobs[24].RunAfter, jobs[24].ID)
	if len(jobs) != 1 {
		t.Errorf("expect ProcessedJobs 1, actual %d", len(jobs))
	}
	if jobs[0].Timeout != 1 {
		t.Errorf("expect timeout 1, actual %d", jobs[0].Timeout)
	}
}

func TestFailedJob(t *testing.T) {
	TruncateJob()

	for i := 1; i <= 51; i++ {
		job := NewJob("test", nil, uint(i))
		job.RunCount = 2
		job.RunAfter = time.Now().Add(-4 * time.Hour)
		job.Save()
		job.Fail("fail")
	}

	jobs, _ := FailedJobs(time.Time{}, 0)
	if len(jobs) != 25 {
		t.Errorf("expect FailedJobs 25, actual %d", len(jobs))
	}
	if jobs[0].Timeout != 51 {
		t.Errorf("expect timeout 51, actual %d", jobs[0].Timeout)
	}
	jobs, _ = FailedJobs(jobs[24].RunAfter, jobs[24].ID)
	if len(jobs) != 25 {
		t.Errorf("expect FailedJobs 25, actual %d", len(jobs))
	}
	if jobs[0].Timeout != 26 {
		t.Errorf("expect timeout 26, actual %d", jobs[0].Timeout)
	}
	jobs, _ = FailedJobs(jobs[24].RunAfter, jobs[24].ID)
	if len(jobs) != 1 {
		t.Errorf("expect FailedJobs 1, actual %d", len(jobs))
	}
	if jobs[0].Timeout != 1 {
		t.Errorf("expect timeout 1, actual %d", jobs[0].Timeout)
	}
}
