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
		return d.handleUpsertHttp(ctx, spec)
	}

	err = fmt.Errorf("daemon sets are currently unsupported for k8s runner types")
	logrus.Error(err)
	return task.Error(err)

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

	// check if the daemon set is running, and get the port
	port, running := d.getPort(spec.Type)
	if !running {
		return task.Error(fmt.Errorf("no daemon set of type [%s] is currently running", spec.Type))
	}

	// check if daemon set is already running a task with the given ID
	_, ok := d.daemonsets[spec.Type].Tasks[spec.DaemonTaskId]
	if ok {
		return task.Error(fmt.Errorf("task with ID [%s] is already running in daemon set of type [%s]", spec.DaemonTaskId, spec.Type))
	}

	logrus.Infof("assigning task [%s] to daemon set of type [%s], running on port [%d]", spec.DaemonTaskId, spec.Type, port)
	daemonTask := DaemonTask{ID: spec.DaemonTaskId, Params: spec.Params}
	_, err = d.client.Assign(ctx, port, &DaemonTasks{Tasks: []DaemonTask{daemonTask}})
	if err != nil {
		return task.Error(err)
	}
	// insert the new task's ID in the daemonset's task set
	d.daemonsets[spec.Type].Tasks[spec.DaemonTaskId] = true
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

	// check if the daemon set is running, and get the port
	port, running := d.getPort(spec.Type)
	if !running {
		return task.Error(fmt.Errorf("no daemon set of type [%s] is currently running", spec.Type))
	}

	// check if daemon set is running a task with the given ID
	_, ok := d.daemonsets[spec.Type].Tasks[spec.DaemonTaskId]
	if !ok {
		return task.Error(fmt.Errorf("no task with ID [%s] is running in daemon set of type [%s]", spec.DaemonTaskId, spec.Type))
	}

	logrus.Infof("removing task [%s] from daemon set of type [%s], running in port [%d]", spec.DaemonTaskId, spec.Type, port)
	_, err = d.client.Remove(ctx, port, &[]string{spec.DaemonTaskId})
	if err != nil {
		return task.Error(err)
	}
	// delete the task's ID from the daemonset's task set
	delete(d.daemonsets[spec.Type].Tasks, spec.DaemonTaskId)
	return task.Respond(&DaemonTaskRemoveResponse{})
}

func (d *Driver) handleUpsertHttp(ctx context.Context, in *DaemonSetUpsertRequest) task.Response {
	d.lock.Lock()
	defer d.lock.Unlock()

	port, running := d.getPort(in.Type)
	if running {
		// check if the config passed in the request is the same as the existing daemon set's
		daemonset := d.daemonsets[in.Type]
		if reflect.DeepEqual(daemonset.Config, in.Config) {
			// If the configs are the same, no need to restart the daemon set
			logrus.Infof("daemon set with id %s of type %s already exists with identical configuration",
				"at port %d. Resetting its id to %s", daemonset.DaemonSetId, in.Type, port,
				in.DaemonSetId)
			daemonset.DaemonSetId = in.DaemonSetId
			return task.Respond(&DaemonSetUpsertResponse{DaemonSetId: in.DaemonSetId, State: StateSuccess})
		}
	}

	// download the daemon set's implementation
	path, err := d.downloader.Download(ctx, in.Config.Repository, in.Config.ExecutableUrls)
	if err != nil {
		logrus.WithError(err).Error("task code download failed")
		return task.Error(err)
	}

	var binpath string
	if *in.Config.ExecutableUrls != nil {
		// if an executable is downloaded directly via url, no need to use `builder`
		binpath = path
	} else {
		// build the daemon set's binary
		builder := builder.New(filepath.Join(path, taskYmlPath))
		binpath, err = builder.Build(ctx)
		if err != nil {
			logrus.WithError(err).Error("task build failed")
			return task.Error(err)
		}
	}

	if running {
		logrus.Infof("daemon set of type %s already exists at port %d. Stopping the process now", in.Type, port)
		err = d.kill(in.Type)
		if err != nil {
			logrus.WithError(err).Errorf("failed to kill daemon set process of type %s running at port %d", in.Type, port)
			return task.Error(err)
		}
		delete(d.daemonsets, in.Type)
	}

	// create the command to run the executable with the -port flag
	cmd := exec.Command(binpath, "-port", fmt.Sprintf("%d", port))

	// set the environment variables
	cmd.Env = append(os.Environ(), in.Config.Envs...)

	// start the command
	if err := cmd.Start(); err != nil {
		logrus.WithError(err).Error("error starting the command")
		return task.Error(err)
	}

	d.daemonsets[in.Type] = &DaemonSet{DaemonSetId: in.DaemonSetId, Config: in.Config, Execution: cmd, Port: port, Tasks: make(map[string]bool)}

	logrus.WithField("path", binpath).
		WithField("port", port).
		WithField("pid", cmd.Process.Pid).
		WithField("type", in.Type).
		Info("started daemon set process")

	return task.Respond(&DaemonSetUpsertResponse{DaemonSetId: in.DaemonSetId, State: StateSuccess})
}

// getPort checks whether a daemon set of given type is running and returns its port
// if not running, returns the port where it should listen when started
func (d *Driver) getPort(t string) (int, bool) {
	daemonset, running := d.daemonsets[t]
	var port int
	if running {
		port = daemonset.Port
	} else {
		port = d.nextPort
		d.nextPort++
	}
	return port, running
}

func (d *Driver) kill(t string) error {
	return d.daemonsets[t].Execution.Process.Kill()
}
