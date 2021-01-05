🎉 Thank you for reporting this! 

**Version number of controlcenter**

Please state version here (found via the "i" info button top right of Web UI)

Version: 

**Postfix version**

Use `postconf mail_version` to get postfix version

Version: 

**Golang version**

Note: needed only if you build Lightmeter Control Center yourself.

Use `go version` to get golang version

Version: 

**Docker version**

Note: "if you use docker then please add the version information"

Use `docker version` to get docker version

Version: 

**Sqlite version**

Use this snippet to extract the version information from your DB files

```
for f in /path/to/workspace/*.db; do
   echo "Filename: $f"
   sqlite3 "$f" '.dbinfo'
 done
```

Version: 

**Attach logs**

Consider attaching relevant mail logs to this issue - you can safely remove private data from them (email addresses, hostnames, IPs, etc)
using the [`tools/batch_log_cleaner.py`] script in this repository. That creates clean log copies without modifying your source logs.

You can run it like this:

```sh
./tools/batch_log_cleaner.py -i /var/log/ -o logs.tar.gz
```

It'll generate a file logs.tar.gz with sample lines (by default 1000 lines) of the Postfix log files in /var/log, removing any sensitive data.

Please run such script on the original log files, or at least on a copy of such files that preserve the file modification time
(copied by rsync, for instance, instead of a plain "cp"), as such metadata is important for analysing the log files.

If you prefer not collecting samples, but sending us all file lines, please add the argument `--complete` to the script execution, as in:

```sh
./tools/batch_log_cleaner.py -i /var/log/ -o logs.tar.gz --complete
```

**Which Operating System and version are you using?**

OS: 

Version:
