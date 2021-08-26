// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package postfix

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bmatsuo/lmdb-go/lmdb"
	"gitlab.com/lightmeter/controlcenter/agent/driver"
	"gitlab.com/lightmeter/controlcenter/agent/parser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"os"
	"strings"
)

const checkFilename = "/etc/postfix/lightmeter_checks"

var checkFilenamePostconfLine = fmt.Sprintf("check_client_access lmdb:%s", checkFilename)

type ipList map[string]struct{}

func buildIPList(ctx context.Context, d driver.Driver) (ipList, error) {
	dbFilename := fmt.Sprintf("%s.lmdb", checkFilename)

	// Does nothing if the lmdb database does not exist, as it'll be created when postmap is called
	if err := d.ExecuteCommand(ctx, []string{"stat", dbFilename}, nil, io.Discard, io.Discard); err != nil {
		return ipList{}, nil
	}

	env, err := lmdb.NewEnv()
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	blockFile, err := os.CreateTemp("", "")
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	tempCopyFilename := blockFile.Name()

	defer func() { _ = os.Remove(tempCopyFilename) }()

	if err := driver.ReadFileContent(ctx, d, dbFilename, blockFile); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err := blockFile.Close(); err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err := env.Open(tempCopyFilename, lmdb.Readonly|lmdb.NoSubdir, 0600); err != nil {
		return nil, errorutil.Wrap(err)
	}

	ipList := ipList{}

	if err := env.View(func(tx *lmdb.Txn) (err error) {
		dbi, err := tx.OpenRoot(0)
		if err != nil {
			return err
		}

		cur, err := tx.OpenCursor(dbi)
		if err != nil {
			return err
		}

		defer cur.Close()

		for {
			k, _, err := cur.Get(nil, nil, lmdb.Next)
			if lmdb.IsNotFound(err) {
				return nil
			}

			if err != nil {
				return err
			}

			keyWithoutNullTerminator := string(bytes.Trim(k, "\x00"))

			ipList[keyWithoutNullTerminator] = struct{}{}
		}
	}); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return ipList, nil
}

func BlockIPs(ctx context.Context, d driver.Driver, ips []string) error {
	blockedIps, err := buildIPList(ctx, d)
	if err != nil {
		return errorutil.Wrap(err)
	}

	for _, ip := range ips {
		blockedIps[ip] = struct{}{}
	}

	// TODO: read ips already in the configuration,
	// to prevent duplicating entries or missing existing ones
	content := bytes.Buffer{}

	// TODO: write the file while storing it, to preventing allocating memory to all of it
	// which is bad if the number of IPs is very large...
	for ip := range blockedIps {
		content.WriteString(fmt.Sprintf("%s REJECT Your IP is Blocked\n", ip))
	}

	tempFilename := checkFilename + ".temp"

	// NOTE: it'll fail if the whole process succeeds, as the file will have been moved
	defer func() { _ = d.ExecuteCommand(ctx, []string{"rm", "-f", tempFilename}, nil, nil, nil) }()

	if err := driver.WriteFileContent(ctx, d, tempFilename, &content); err != nil {
		return errorutil.Wrap(err)
	}

	if err := updatePostfixConfigIfNeeded(ctx, d); err != nil {
		return errorutil.Wrap(err)
	}

	if err := d.ExecuteCommand(ctx, []string{"mv", tempFilename, checkFilename}, nil, driver.Stdout, driver.Stderr); err != nil {
		return errorutil.Wrap(err)
	}

	// generate database
	if err := d.ExecuteCommand(ctx, []string{"postmap", checkFilename}, nil, driver.Stdout, driver.Stderr); err != nil {
		return errorutil.Wrap(err)
	}

	// tell postfix to reload the configuration
	if err := d.ExecuteCommand(ctx, []string{"postfix", "reload"}, nil, driver.Stdout, driver.Stderr); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func getPostconf(ctx context.Context, d driver.Driver, args ...string) (*parser.Parser, error) {
	stdout := bytes.Buffer{}

	if err := d.ExecuteCommand(ctx, append([]string{"postconf"}, args...), nil, &stdout, driver.Stderr); err != nil {
		return nil, errorutil.Wrap(err)
	}

	conf, err := parser.Parse(stdout.Bytes())
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return conf, nil
}

func updatePostfixConfigIfNeeded(ctx context.Context, d driver.Driver) error {
	// here we get the config resolving the values (-x), as the resl value might be in a variable
	conf, err := getPostconf(ctx, d, "-x")
	if err != nil {
		return errorutil.Wrap(err)
	}

	restrictions, err := conf.Value("smtpd_recipient_restrictions")
	if err != nil {
		return errorutil.Wrap(err)
	}

	index := strings.Index(restrictions, checkFilenamePostconfLine)
	if index != -1 {
		// setting already set. Nothing to do here
		return nil
	}

	// postfix should be setup
	if err := setupPostfixRestrictions(ctx, d); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func setupPostfixRestrictions(ctx context.Context, d driver.Driver) error {
	// here we get the config without resolving the values, to preserve its original content
	conf, err := getPostconf(ctx, d)
	if err != nil {
		return errorutil.Wrap(err)
	}

	restrictions, err := conf.Value("smtpd_recipient_restrictions")
	if err != nil {
		return errorutil.Wrap(err)
	}

	// prepend the file to make it have higher priority over other checks
	setCommand := fmt.Sprintf(`smtpd_recipient_restrictions=%s, %s`, checkFilenamePostconfLine, restrictions)

	if err := d.ExecuteCommand(ctx, []string{`postconf`, setCommand}, nil, driver.Stdout, driver.Stderr); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
