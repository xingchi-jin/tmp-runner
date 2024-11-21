# runner

Runner executes tasks from Harness.

Run it with

```
go build
./runner server --env-file local_runner.env
```

To run a sample local execution, run the following commands

```
cd tasks/sample
go build
./sample
```

To generate wire dependencies, run wire in cli/ folder:
```
cd cli
wire
```
