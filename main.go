package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"text/tabwriter"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Job struct {
	Id          int
	ProcessId   int
	Name        string
	Command     string
	CreatedAt   string
	FinnishedAt string
}

var db *sql.DB

// GetConnection returns the database connection and initializes the schema if needed
func GetConnection() *sql.DB {
	if db != nil {
		return db
	}
	var err error
	db, err = sql.Open("sqlite3", "jobs.db")
	if err != nil {
		panic(err)
	}

	// Create table if it doesn't exist yet
	schema := `
	CREATE TABLE IF NOT EXISTS job (
		ID INTEGER PRIMARY KEY,
		PROCESS_ID INTEGER,
		NAME TEXT,
		COMMAND TEXT,
		CREATED_AT TEXT,
		FINNISHED_AT TEXT
	);`
	_, err = db.Exec(schema)
	if err != nil {
		panic(err)
	}

	return db
}

func (j Job) Create() error {
	db := GetConnection()
	query := `INSERT INTO job (ID, PROCESS_ID, NAME, COMMAND, CREATED_AT, FINNISHED_AT) VALUES (?, ?, ?, ?, ?, ?)`
	stmt, err := db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	r, err := stmt.Exec(j.Id, j.ProcessId, j.Name, j.Command, j.CreatedAt, "")
	if err != nil {
		return err
	}
	if i, err := r.RowsAffected(); err != nil || i != 1 {
		return errors.New("ERROR: at least one row was expected to be affected")
	}
	return nil
}

func DeleteJob(id int) error {
	db := GetConnection()
	q := `DELETE FROM job WHERE ID=?`
	stmt, err := db.Prepare(q)
	if err != nil {
		return err
	}
	defer stmt.Close()
	r, err := stmt.Exec(id)
	if err != nil {
		return err
	}
	if i, err := r.RowsAffected(); err != nil || i != 1 {
		return errors.New("ERROR: job not found or couldn't be deleted")
	}
	return nil
}

func UpdateJobFinished(id int) error {
	db := GetConnection()
	q := `UPDATE job SET FINNISHED_AT = ? WHERE ID=?`
	stmt, err := db.Prepare(q)
	if err != nil {
		return err
	}
	defer stmt.Close()
	r, err := stmt.Exec(time.Now().Format("2006-01-02 15:04:05"), id)
	if err != nil {
		return err
	}
	if i, err := r.RowsAffected(); err != nil || i != 1 {
		return errors.New("ERROR: job not found or couldn't be updated")
	}
	return nil
}

func CancelJob(jobId int) {
	db := GetConnection()

	// 1. Get the PID associated with this Job ID
	var pid int
	var command string
	err := db.QueryRow("SELECT PROCESS_ID, COMMAND FROM job WHERE ID = ?", jobId).Scan(&pid, &command)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("Job with ID %d not found.\n", jobId)
		} else {
			fmt.Printf("Database error: %v\n", err)
		}
		return
	}

	fmt.Printf("Canceling job %d (PID: %d, Command: '%s')...\n", jobId, pid, command)

	// 2. Find and kill the process
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("Could not find process with PID %d: %v\n", pid, err)
		return
	}

	// Send kill signal
	err = process.Kill()
	if err != nil {
		fmt.Printf("Failed to kill process: %v\n", err)
		return
	}

	// 3. Mark the job as finished/stopped in DB
	_ = UpdateJobFinished(jobId)
	fmt.Println("Job successfully canceled and updated in database.")
}

func ListJobs() {
	db := GetConnection()
	rows, err := db.Query("SELECT ID, PROCESS_ID, NAME, COMMAND, CREATED_AT, FINNISHED_AT FROM job")
	if rows.Err() != nil {
		log.Fatalf("Failed to fetch jobs: %v", err)
	}
	defer rows.Close()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(w, "ID\tPID\tNAME\tCOMMAND\tCREATED AT\tFINISHED AT")
	fmt.Fprintln(w, "--\t---\t----\t-------\t----------\t-----------")

	for rows.Next() {
		var j Job
		err := rows.Scan(&j.Id, &j.ProcessId, &j.Name, &j.Command, &j.CreatedAt, &j.FinnishedAt)
		if err != nil {
			log.Fatal(err)
		}
		if j.FinnishedAt == "" {
			j.FinnishedAt = "RUNNING"
		}
		fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\t%s\n", j.Id, j.ProcessId, j.Name, j.Command, j.CreatedAt, j.FinnishedAt)
	}
	w.Flush()
}

func main() {
	// CLI Flags
	jobName := flag.String("job_name", "test_job", "Name of the job")
	commandPtr := flag.String("add_job", "", "Command to run as a background job")
	cancelId := flag.Int("cancel", 0, "ID of the job to cancel")
	listJobs := flag.Bool("list", false, "List all managed jobs")

	flag.Parse()

	// Ensure DB connections are cleaned up when program closes
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	// Route 1: List Jobs
	if *listJobs {
		ListJobs()
		return
	}

	// Route 2: Cancel Job
	if *cancelId != 0 {
		CancelJob(*cancelId)
		return
	}

	// Route 3: Add Job
	if *commandPtr != "" {
		if *jobName == "" {
			fmt.Println("Please provide a job name with -job_name")
			return
		}

		job := Job{
			Id:          rand.IntN(100000), // Larger ceiling to prevent easy collisions
			Name:        *jobName,
			Command:     *commandPtr,
			CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
			FinnishedAt: "",
		}

		// Runs command, and updates DB automatically when finished using sqlite3 CLI
		// We use standard time formatting inside SQLite to avoid missing timezone issues
		cmdStr := fmt.Sprintf("%s; sqlite3 jobs.db \"UPDATE job SET FINNISHED_AT = datetime('now', 'localtime') WHERE ID = %d;\"", job.Command, job.Id)
		cmd := exec.Command("/bin/bash", "-c", cmdStr)

		// Detach stdout/stderr so background execution doesn't block terminal outputs
		cmd.Stdout = nil
		cmd.Stderr = nil

		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}

		job.ProcessId = cmd.Process.Pid
		err := job.Create()
		if err != nil {
			fmt.Printf("Error logging job to database: %v\n", err)
			return
		}

		fmt.Printf("Successfully started job '%s' (ID: %d, PID: %d) in the background.\n", job.Name, job.Id, job.ProcessId)
		return
	}

	// Default Fallback
	fmt.Println("Welcome to TicTac Job Manager")
	fmt.Println("Usage:")
	fmt.Println("  Add a job:    go run main.go -add_job \"sleep 10\" -job_name \"MySleepJob\"")
	fmt.Println("  List jobs:    go run main.go -list")
	fmt.Println("  Cancel job:   go run main.go -cancel <job_id>")
}
