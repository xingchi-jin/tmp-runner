package daemonset

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/drone/go-task/task"
	"github.com/drone/go-task/task/builder"
	"github.com/drone/go-task/task/download"
	"github.com/sirupsen/logrus"
)

var (
	taskYmlPath = "task.yml"
)

type Driver struct {
	client     *Client
	downloader download.Downloader
	daemonsets map[string]*DaemonSet
	isK8s      bool
	lock       sync.Mutex
	nextPort   int
}

// New returns the daemon set task execution driver
func New(d download.Downloader, isK8s bool) *Driver {
	return &Driver{client: newClient(), downloader: d, daemonsets: make(map[string]*DaemonSet), isK8s: isK8s, nextPort: 9000}
}

// HandleUpsert handles upserting a daemon set process
func (d *Driver) HandleUpsert(ctx context.Context, req *task.Request) task.Response {
	spec := new(DaemonSetUpsertRequest)
	err := json.Unmarshal(req.Task.Data, spec)
	if err != nil {
		return task.Error(err)
	}

	if !d.isK8s {
		// for now, we only support daemon sets for non-k8s runner types
		return d.upsertHttp(ctx, spec)
	}

	err = fmt.Errorf("daemon sets are currently unsupported for k8s runner types")
	logrus.Error(err)
	return task.Respond(&DaemonSetUpsertResponse{DaemonSetId: spec.DaemonSetId, State: StateFailure, Error: err.Error()})

}

// HandleTaskAssign handles assigning a new daemon task to a daemon set
func (d *Driver) HandleTaskAssign(ctx context.Context, req *task.Request) task.Response {
	spec := new(DaemonTaskAssignRequest)
	err := json.Unmarshal(req.Task.Data, spec)
	if err != nil {
		return task.Error(err)
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	// check if the daemon set is running
	ds, running := d.daemonsets[spec.Type]
	if !running {
		errMsg := fmt.Sprintf("no daemon set of type [%s] is currently running", spec.Type)
		return task.Respond(&DaemonTaskAssignResponse{DaemonTaskId: spec.DaemonTaskId, State: StateFailure, Error: errMsg})
	}

	// check if daemon set is already running a task with the given ID
	_, ok := ds.Tasks[spec.DaemonTaskId]
	if ok {
		errMsg := fmt.Sprintf("task with id [%s] is already running in daemon set of type [%s]", spec.DaemonTaskId, spec.Type)
		return task.Respond(&DaemonTaskAssignResponse{DaemonTaskId: spec.DaemonTaskId, State: StateFailure, Error: errMsg})
	}

	dsLogger(ds).Infof("assigning task [%s] to daemon set", spec.DaemonTaskId)
	daemonTask := DaemonTask{ID: spec.DaemonTaskId, Params: spec.Params}
	_, err = d.client.Assign(ctx, ds.Port, &DaemonTasks{Tasks: []DaemonTask{daemonTask}})
	if err != nil {
		return task.Error(err)
	}
	// insert the new task's ID in the daemonset's task set
	ds.Tasks[spec.DaemonTaskId] = true
	return task.Respond(&DaemonTaskAssignResponse{DaemonTaskId: spec.DaemonTaskId, State: StateSuccess})
}

// HandleTaskRemove handles removing a daemon task from a daemon set
func (d *Driver) HandleTaskRemove(ctx context.Context, req *task.Request) task.Response {
	spec := new(DaemonTaskRemoveRequest)
	err := json.Unmarshal(req.Task.Data, spec)
	if err != nil {
		return task.Error(err)
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	// check if the daemon set is running
	ds, running := d.daemonsets[spec.Type]
	if !running {
		errMsg := fmt.Sprintf("no daemon set of type [%s] is currently running", spec.Type)
		return task.Respond(&DaemonTaskRemoveResponse{DaemonTaskId: spec.DaemonTaskId, State: StateFailure, Error: errMsg})
	}

	// check if daemon set is running a task with the given ID
	_, ok := ds.Tasks[spec.DaemonTaskId]
	if !ok {
		errMsg := fmt.Sprintf("no task with id [%s] is running in daemon set of type [%s]", spec.DaemonTaskId, spec.Type)
		return task.Respond(&DaemonTaskRemoveResponse{DaemonTaskId: spec.DaemonTaskId, State: StateFailure, Error: errMsg})
	}

	dsLogger(ds).Infof("removing task [%s] from daemon set", spec.DaemonTaskId)
	_, err = d.client.Remove(ctx, ds.Port, &[]string{spec.DaemonTaskId})
	if err != nil {
		return task.Error(err)
	}
	// delete the task's ID from the daemonset's task set
	delete(ds.Tasks, spec.DaemonTaskId)
	return task.Respond(&DaemonTaskRemoveResponse{DaemonTaskId: spec.DaemonTaskId, State: StateSuccess})
}

// upsertHttp will handle upserting a daemon set process that runs as http server
func (d *Driver) upsertHttp(ctx context.Context, in *DaemonSetUpsertRequest) task.Response {
	d.lock.Lock()
	defer d.lock.Unlock()

	if runningWithIdenticalConfig := d.handleRunningWithSameConfig(in); runningWithIdenticalConfig {
		return task.Respond(&DaemonSetUpsertResponse{DaemonSetId: in.DaemonSetId, State: StateSuccess})
	}

	path, err := d.download(ctx, in)
	if err != nil {
		return task.Respond(&DaemonSetUpsertResponse{DaemonSetId: in.DaemonSetId, State: StateFailure, Error: err.Error()})
	}

	binpath, err := d.build(ctx, in, path)
	if err != nil {
		return task.Respond(&DaemonSetUpsertResponse{DaemonSetId: in.DaemonSetId, State: StateFailure, Error: err.Error()})
	}

	if err = d.handleRunningWithDifferentConfig(in); err != nil {
		return task.Respond(&DaemonSetUpsertResponse{DaemonSetId: in.DaemonSetId, State: StateFailure, Error: err.Error()})
	}

	port := d.getPort(in.Type)

	cmd, err := d.startProcess(in, binpath, port)
	if err != nil {
		return task.Respond(&DaemonSetUpsertResponse{DaemonSetId: in.DaemonSetId, State: StateFailure, Error: err.Error()})
	}

	ds := &DaemonSet{DaemonSetId: in.DaemonSetId, Type: in.Type, Config: in.Config, Execution: cmd, Port: port, Tasks: make(map[string]bool)}
	d.daemonsets[in.Type] = ds

	dsLogger(ds).Info("started daemon set process")

	return task.Respond(&DaemonSetUpsertResponse{DaemonSetId: in.DaemonSetId, State: StateSuccess})
}

// check if the daemon set is already running with same config as requested
// if this is the case, set the currently running daemon set's ID to the one passed in the request, and return true
// otherwise, return false
func (d *Driver) handleRunningWithSameConfig(in *DaemonSetUpsertRequest) bool {
	ds, running := d.daemonsets[in.Type]
	if running {
		// check if the config passed in the request is the same as the existing daemon set's
		if reflect.DeepEqual(ds.Config, in.Config) {
			// If the configs are the same, no need to restart the daemon set
			dsLogger(ds).Infof("daemon set of type [%s] is running with identical configuration. "+
				"Resetting its id to [%s]", in.Type, in.DaemonSetId)
			ds.DaemonSetId = in.DaemonSetId
			return true
		}
	}
	return false
}

// check if the daemon set is already running with config different from requested
// if this is the case, kill the current daemon set process
func (d *Driver) handleRunningWithDifferentConfig(in *DaemonSetUpsertRequest) error {
	ds, running := d.daemonsets[in.Type]
	if running {
		dsLogger(ds).Infof("daemon set of type [%s] is running. Stopping the process now", in.Type)
		err := d.kill(in.Type)
		if err != nil {
			dsLogger(ds).WithError(err).Error("failed to kill daemon set process")
			return err
		}
		delete(d.daemonsets, in.Type)
	}
	return nil
}

// download the daemon set's repository or executable file
func (d *Driver) download(ctx context.Context, in *DaemonSetUpsertRequest) (string, error) {
	path, err := d.downloader.Download(ctx, in.Type, in.Config.Repository, in.Config.Executable)
	if err != nil {
		logrus.WithError(err).Error("task code download failed")
		return "", err
	}
	return path, nil
}

// build the daemon set's executable and returns its full path
func (d *Driver) build(ctx context.Context, in *DaemonSetUpsertRequest, path string) (string, error) {
	if in.Config.Executable != nil {
		// if an executable is downloaded directly via url, no need to use `builder`
		return path, nil
	}
	// build the daemon set's binary
	builder := builder.New(filepath.Join(path, taskYmlPath))
	binpath, err := builder.Build(ctx)
	if err != nil {
		logrus.WithError(err).Error("task build failed")
		return "", err
	}
	return binpath, nil
}

// spawns daemon set process passing it the -port param
func (d *Driver) startProcess(in *DaemonSetUpsertRequest, binpath string, port int) (*exec.Cmd, error) {
	// create the command to run the executable with the -port flag
	cmd := exec.Command(binpath, "-port", fmt.Sprintf("%d", port))

	// set the environment variables
	cmd.Env = append(os.Environ(), in.Config.Envs...)

	// start the command
	if err := cmd.Start(); err != nil {
		logrus.WithError(err).Error("error starting the command")
		return nil, err
	}
	return cmd, nil
}

// getPort checks whether a daemon set of given type is running and returns its port
// if not running, returns the port where it should listen when started
func (d *Driver) getPort(t string) int {
	daemonset, running := d.daemonsets[t]
	var port int
	if running {
		port = daemonset.Port
	} else {
		port = d.nextPort
		d.nextPort++
	}
	return port
}

// kill will stop a daemon set process, given the daemon set's type
func (d *Driver) kill(t string) error {
	return d.daemonsets[t].Execution.Process.Kill()
}

// returns a logrus *Entry with daemon set's data as fields
func dsLogger(ds *DaemonSet) *logrus.Entry {
	return logrus.WithField("id", ds.DaemonSetId).
		WithField("type", ds.Type).
		WithField("port", ds.Port).
		WithField("pid", ds.Execution.Process.Pid).
		WithField("binpath", ds.Execution.Path)

}
