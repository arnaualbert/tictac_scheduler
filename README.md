# tictac_scheduler

Building a scheduler in Go

# TODO

- [ ] Make it a CLI tool

- [ ] Create the queues

- [ ] Document and create the page

- [ ] Create the installation script

- [ ] Add job owner to the stuct and db

# Stack:

- Go
- Sqlite

# Job info

- Id is a random integer

- ProcessId is the process id of the task

- Name is the id that the user will put to recognise easier the job

- Command is the task that will be excecuted

- CreatedAt is the date when the task start

- FinnishedAt is the date when the task ends

```go
type Job struct {
	Id          int
	ProcessId   int
	Name        string
	Command     string
	CreatedAt   string
	FinnishedAt string
}
```