🎉 Thank you for reporting this!

Please fill the info below, skipping when not available.

**Version number of Lightmeter**

Please state version here (found via the "i" info button top right of Web UI) or by running Lightmeter with `-version`:

Version:

**Postfix version**

Use `postconf mail_version` to get postfix version

Version:

**Golang version**

In case you have built Lightmeter yourself.

Use `go version` to get golang version

Version:

**Docker version**

Note: "if you use docker then please add the version information"

Use `docker version` to get docker version

Version:

**Which Operating System and version are you using?**

OS:

Version:

**Attach logs**

Please consider attaching relevant mail logs to this issue - you can safely remove private data from them (email addresses, hostnames, IPs, etc)
using the [`tools/batch_log_cleaner.py`] script in this repository. That creates clean log copies without modifying your source logs.

You can run it like this:

```sh
./tools/batch_log_cleaner.py -i /var/log/ -o logs.tar.gz --complete
```

It'll generate a file logs.tar.gz with sample lines (by default 1000 lines) of the Postfix log files in /var/log, removing any sensitive data.

Please run such script on the original log files, or at least on a copy of such files that preserve the file modification time
(copied by rsync, for instance, instead of a plain "cp"), as such metadata is important for analysing the log files.

Please always check the contents of the `logs.tar.gz` file before sending it to us :-)

You can attach the `logs.tar.gz` file to this issue, or send it via e-mail to `hello@lightmeter.io` with the subject
`Log files for Gitlab issue #XXX`, where `#XXX` is the number assigned to the issue report you've just created
(you can copy if from the URL or from the top of the page).
