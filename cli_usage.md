```
Lightmeter ControlCenter 1.5.0

 Example call: 

 ./lightmeter -workspace ~/lightmeter_workspace -watch_dir /var/log 

 Flag set: 

  -email_reset string
    	Reset password for user (implies -password and depends on -workspace)
  -importonly
    	Only import existing logs, exiting immediately, without running the full application.
  -listen string
    	Network address to listen to (default ":8080")
  -log_format string
    	Expected log format from external sources (like logstash, etc.) (default "default")
  -log_starting_year int
    	Value to be used as initial year when it cannot be obtained from the Postfix logs. Defaults to the current year. Requires -stdin.
  -migrate_down_to_database string
    	Database name only for migration
  -migrate_down_to_only
    	Only migrates down
  -migrate_down_to_version int
    	Specify the new migration version (default -1)
  -password string
    	Password to reset (requires -email_reset)
  -rsync
    	Log directory is updated by rsync
  -socket string
    	Receive logs via a socket. E.g. unix=/tmp/lightemter.sock or tcp=localhost:9999
  -stdin
    	Read log lines from stdin
  -verbose
    	Be Verbose
  -version
    	Show Version Information
  -watch_dir string
    	Path to the directory where postfix stores its log files, to be watched
  -workspace string
    	Path to the directory to store all working data (default "/var/lib/lightmeter_workspace")
```
