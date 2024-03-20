package tasks

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

var (
	ctxBg = context.Background()
)

func main() {
	fmt.Println("Hello, World!")
	stageID := uuid.New().String()
	// setup
	req := exampleSetupRequest(stageID)
	err := HandleSetup(ctxBg, req)
	if err != nil {
		panic(err)
	}

	// execute two steps
	step1ID := uuid.New().String()
	q := sampleExecRequest(step1ID, stageID, []string{"echo", "hello"})
	resp, err := HandleExec(ctxBg, q)
	if err != nil {
		panic(err)
	}
	fmt.Printf("poll response: %+v", resp)

	step2ID := uuid.New().String()
	r := sampleExecRequest(step2ID, stageID, []string{"sleep", "10"})
	resp, err = HandleExec(ctxBg, r)
	if err != nil {
		panic(err)
	}
	fmt.Printf("poll response: %+v", resp)

	// cleanup
	d := sampleDestroyRequest(stageID)
	err = HandleDestroy(ctxBg, d)
	if err != nil {
		panic(err)
	}

	fmt.Println("successfully completed!")

}
