package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/harness/lite-engine/logstream"
	"github.com/harness/runner/tasks"
)

var (
	ctxBg = context.Background()
)

func main() {
	fmt.Println("Hello, World!")
	stageID := uuid.New().String()
	// setup
	req := tasks.SampleSetupRequest(stageID)
	_, err := tasks.HandleSetup(ctxBg, &req, "")
	if err != nil {
		panic(err)
	}

	// execute two steps
	step1ID := uuid.New().String()
	q := tasks.SampleExecRequest(step1ID, stageID, []string{"touch a.txt"}, "", []string{"sh", "-c"})
	resp, err := tasks.HandleExec(ctxBg, q, logstream.NopWriter())
	if err != nil {
		panic(err)
	}
	fmt.Printf("poll response: %+v", resp)

	step2ID := uuid.New().String()
	r := tasks.SampleExecRequest(step2ID, stageID, []string{"ls"}, "alpine", []string{"sh", "-c"})
	resp, err = tasks.HandleExec(ctxBg, r, logstream.NopWriter())
	if err != nil {
		panic(err)
	}
	fmt.Printf("poll response: %+v", resp)

	// cleanup
	d := tasks.SampleDestroyRequest(stageID)
	_, err = tasks.HandleDestroy(ctxBg, d)
	if err != nil {
		panic(err)
	}

	fmt.Println("successfully completed!")

}
