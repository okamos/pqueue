package pqueue

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	validator "gopkg.in/go-playground/validator.v9"
)

// JobConfig configurations for job
type JobConfig struct {
	MaxRetryCount uint
	RetryDelay    uint
}

var jobConfig = JobConfig{
	MaxRetryCount: 3,
	RetryDelay:    5,
}

// SetJobConfig sets the configuration about job default values.
func SetJobConfig(j JobConfig) {
	jobConfig = j
}

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Job describes a job in a queue.
type Job struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name" validate:"required"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	Status     uint            `json:"status" validate:"gte=0,lte=2"` // 0 yet, 1 processed, 2 failed
	Priority   int             `json:"priority"`
	RunAfter   time.Time       `json:"run_after"`
	Timeout    uint            `json:"time_out" validate:"gt=0"`
	RunCount   uint            `json:"run_count"`
	RetryDelay uint            `json:"retry_delay"` // second
	Elapsed    float64         `json:"elapsed"`
	LastError  string          `json:"last_error"`
}

// NewJob creates a job. NOTE: timeout should be greater than 0.
func NewJob(name string, payload json.RawMessage, timeout uint) Job {
	return Job{
		Name:       name,
		Payload:    payload,
		Status:     0, // yet
		Priority:   0,
		RunAfter:   time.Now(),
		Timeout:    timeout,
		RunCount:   0,
		RetryDelay: jobConfig.RetryDelay,
	}
}

// Save inserts a job.
func (j *Job) Save() error {
	err := validate.Struct(j)
	if err != nil {
		return err
	}
	var payload interface{}
	if len(j.Payload) > 0 {
		err = json.Unmarshal(j.Payload, &payload)
		if err != nil {
			return err
		}
	}

	stmt, err := db.Prepare(`INSERT INTO "job" (name,payload,status,priority,run_after,timeout,run_count,retry_delay) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`)
	if err != nil {
		return err
	}

	err = stmt.QueryRow(
		j.Name,
		j.Payload,
		j.Status,
		j.Priority,
		j.RunAfter,
		j.Timeout,
		j.RunCount,
		j.RetryDelay,
	).Scan(&j.ID)
	return err
}

// LockJobs locks rows using advisory lock and returns jobs.
func LockJobs(length int) ([]Job, error) {
	rows, err := db.Query(`UPDATE "job" SET grabbed = now() WHERE id IN (SELECT id FROM (SELECT id FROM "job" WHERE grabbed is NULL AND run_after <= now() AND status = 0 ORDER BY priority desc LIMIT $1) potential_jobs WHERE pg_try_advisory_lock(id)) AND grabbed is NULL RETURNING id, name, payload, run_after, timeout, run_count, retry_delay`, length)

	defer rows.Close()

	if err != nil {
		return nil, err
	}

	var jobs []Job
	for rows.Next() {
		j := Job{}
		err := rows.Scan(
			&j.ID,
			&j.Name,
			&j.Payload,
			&j.RunAfter,
			&j.Timeout,
			&j.RunCount,
			&j.RetryDelay,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// UnlockJobs unlocks rows about
func UnlockJobs() error {
	_, err := db.Exec(`UPDATE "job" SET grabbed = null WHERE id IN (SELECT id FROM (SELECT id FROM "job" WHERE grabbed is NOT NULL AND run_after <= now() AND status = 0) potential_jobs WHERE pg_advisory_unlock(id)) AND grabbed is NOT NULL`)
	return err
}

// ReleaseJobs set grabbed = null, which status = 0
func ReleaseJobs() error {
	_, err := db.Query(`UPDATE "job" SET grabbed = null WHERE id IN (SELECT id FROM (SELECT id FROM "job" WHERE grabbed is NOT NULL AND run_after <= now() AND status = 0) potential_jobs WHERE pg_try_advisory_lock(id)) AND grabbed IS NOT NULL`)
	return err
}

// Complete done a job, or re-queue a job if failed
func (j *Job) Complete() {
	stmt, err := db.Prepare(`UPDATE "job" SET status = 1, run_count = $2, elapsed = $3 WHERE ID = $1 RETURNING pg_advisory_unlock($1)`)
	if err != nil {
		log.Print(err)
		return
	}

	_, err = stmt.Exec(j.ID, j.RunCount+1, j.Elapsed)
	if err != nil {
		log.Print(err)
		return
	}
	j.Status = 1
	j.RunCount++

	log.Printf("Processed job id: %d, name: %s, payload: %s", j.ID, j.Name, j.Payload)
}

// Fail re-queues a job, or makes failed status if run count greater than max retries.
func (j *Job) Fail(errStr string) {
	runCount := j.RunCount + 1

	if runCount >= jobConfig.MaxRetryCount {
		stmt, err := db.Prepare(`UPDATE "job" SET status = 2, run_count = $2, elapsed = $3, last_error = $4 WHERE id = $1 RETURNING pg_advisory_unlock($1)`)
		if err != nil {
			log.Print(err)
			return
		}

		_, err = stmt.Exec(j.ID, runCount, j.Elapsed, errStr)
		if err != nil {
			log.Print(err)
			return
		}
		j.Status = 2
	} else {
		delay := runCount*runCount*runCount*runCount + j.Timeout + j.RetryDelay + 15
		stmt, err := db.Prepare(`UPDATE "job" SET run_count = $2, retry_delay = $3, run_after = $4, elapsed = $5, last_error = $6, grabbed = null WHERE id = $1 RETURNING pg_advisory_unlock($1)`)
		if err != nil {
			log.Print(err)
			return
		}

		_, err = stmt.Exec(
			j.ID,
			runCount,
			delay,
			j.RunAfter.Add(time.Duration(delay)*time.Second),
			j.Elapsed,
			errStr,
		)
		if err != nil {
			log.Print(err)
			return
		}
		j.RetryDelay = delay
		j.RunAfter = j.RunAfter.Add(time.Duration(delay) * time.Second)
	}
	j.RunCount++
	log.Printf("Failed job id: %d, name: %s, payload: %s", j.ID, j.Name, j.Payload)
}

// ProcessingJobs returns jobs, which status is done
func ProcessingJobs() ([]Job, error) {
	rows, err := db.Query(`SELECT id, name, payload, status, priority, run_after, timeout, run_count FROM "job" WHERE status = 0 AND grabbed is not null`)
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	var jobs []Job
	for rows.Next() {
		j := Job{}
		err := rows.Scan(
			&j.ID,
			&j.Name,
			&j.Payload,
			&j.Status,
			&j.Priority,
			&j.RunAfter,
			&j.Timeout,
			&j.RunCount,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// ProcessedJobs returns jobs, which status is done
func ProcessedJobs(prevTime time.Time, prevID int64) ([]Job, error) {
	var rows *sql.Rows
	var err error
	if prevTime.IsZero() {
		query := `SELECT id, name, payload, status, priority, run_after, timeout, run_count, elapsed, last_error FROM "job" WHERE status = 1 ORDER BY run_after desc, id desc limit 25`
		rows, err = db.Query(query)
	} else {
		query := `SELECT id, name, payload, status, priority, run_after, timeout, run_count, elapsed, last_error FROM "job" WHERE status = 1 AND (run_after, id) < ($1, $2) ORDER BY run_after desc, id desc limit 25`
		rows, err = db.Query(query, prevTime, prevID)
	}
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	var jobs []Job
	for rows.Next() {
		j := Job{}
		err := rows.Scan(
			&j.ID,
			&j.Name,
			&j.Payload,
			&j.Status,
			&j.Priority,
			&j.RunAfter,
			&j.Timeout,
			&j.RunCount,
			&j.Elapsed,
			&j.LastError,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// FailedJobs returns jobs, which status is failed
func FailedJobs(prevTime time.Time, prevID int64) ([]Job, error) {
	var rows *sql.Rows
	var err error
	if prevTime.IsZero() {
		query := `SELECT id, name, payload, status, priority, run_after, timeout, run_count, elapsed, last_error FROM "job" WHERE status = 2 ORDER BY run_after desc, id desc limit 25`
		rows, err = db.Query(query)
	} else {
		query := `SELECT id, name, payload, status, priority, run_after, timeout, run_count, elapsed, last_error FROM "job" WHERE status = 2 AND (run_after, id) < ($1, $2) ORDER BY run_after desc, id desc limit 25`
		rows, err = db.Query(query, prevTime, prevID)
	}
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	var jobs []Job
	for rows.Next() {
		j := Job{}
		err := rows.Scan(
			&j.ID,
			&j.Name,
			&j.Payload,
			&j.Status,
			&j.Priority,
			&j.RunAfter,
			&j.Timeout,
			&j.RunCount,
			&j.Elapsed,
			&j.LastError,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
