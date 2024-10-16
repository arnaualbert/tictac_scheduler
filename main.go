package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
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

func GetConnection() *sql.DB {

	if db != nil {
		return db
	}
	var err error
	db, err = sql.Open("sqlite3", "jobs.db")
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

func (j Job) Delete(id int) error {
	db := GetConnection()
	q := `DELETE FROM job
            WHERE id=?`
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
		return errors.New("ERROR: at least one row was expected to be affected")
	}
	return nil
}

func (j Job) Update(id int) error {
	db := GetConnection()
	q := `UPDATE job
			SET FINNISHED_AT = ?
			WHERE id=?`
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
		return errors.New("ERROR: at least one row was expected to be affected")
	}
	return nil
}

func CancelJob() {
	// TODO: Cancel the job by the job id/job name still pending to decide
	fmt.Println("Canceling job")

}

func main() {
	fmt.Println("Welcome to TicTac job manager")
	fmt.Print(time.Now().Format("02-01-2006 15:04:05 Monday\n"))
	job := Job{
		Id:          1,
		Name:        "test",
		Command:     "sleep 10",
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		FinnishedAt: "",
	}
	cmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s; sqlite3 jobs.db 'UPDATE job SET FINNISHED_AT = CURRENT_TIMESTAMP WHERE id = %d;'", job.Command, job.Id))
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil { // run in the background
		log.Fatal(err)
	}
	log.Printf("Just ran subprocess %d, exiting\n", cmd.Process.Pid)
	//
	job.ProcessId = cmd.Process.Pid
	err := job.Create()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Job created")
}
