package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/harness/lite-engine/logstream"
	"github.com/harness/runner/logger"
	"github.com/harness/runner/tasks/local"
)

var (
	ctxBg = context.Background()
)

func main() {
	fmt.Println("Running a sample CI pipeline...")
	stageID := uuid.New().String()
	// setup
	fmt.Println("Setting up the pipeline")
	req := SampleSetupRequest(stageID)
	_, err := local.HandleSetup(ctxBg, &req, "", logstream.NopWriter()) // setup pipeline
	if err != nil {
		panic(err)
	}

	// execute two steps
	fmt.Println("Executing step1")
	step1ID := uuid.New().String()
	q := SampleExecRequest(step1ID, stageID, []string{"touch a.txt"}, "", []string{"sh", "-c"}) // create file on host
	resp, err := local.HandleExec(ctxBg, &q, logstream.NopWriter())
	if err != nil {
		panic(err)
	}
	fmt.Printf("poll response: %+v", resp)

	fmt.Println("Executing step2")
	step2ID := uuid.New().String()
	r := SampleExecRequest(step2ID, stageID, []string{"ls"}, "alpine", []string{"sh", "-c"}) // view files created in container
	resp, err = local.HandleExec(ctxBg, &r, logger.NewWriterWrapper(os.Stdout))
	if err != nil {
		panic(err)
	}
	fmt.Printf("poll response: %+v", resp)

	// cleanup
	fmt.Println("Cleaning up resources")
	d := SampleDestroyRequest(stageID)
	_, err = local.HandleDestroy(ctxBg, d)
	if err != nil {
		panic(err)
	}

	fmt.Println("successfully completed!")

}
