#!/usr/bin/env python3

# SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
#
# SPDX-License-Identifier: AGPL-3.0-only

# -*- coding:utf-8 -*-

"""
To add this script as a pre-commit git hook, simply run:
chmod +x clean_files_go_js_vue.py
pushd .git/hooks/
ln -s ../../clean_files_go_js_vue.py pre-commit
popd
"""


import subprocess, re, os, sys


git_output = subprocess.run(['git','status','--porcelain'], capture_output=True, text=True)

files = filter(lambda f: f!='', re.split('\n', str(git_output.stdout)))

files = map(lambda f: re.sub('^..\s', '', f), files)

files = set(files)

retval = 0


def run_and_check(cmd):
	print("\n",cmd)
	res = subprocess.run(cmd)
	if res.returncode != 0:
		print("[ERROR] Command exited with return code",res.returncode)
	return res.returncode


print("\n##### vue/js files #####")

os.chdir('frontend/controlcenter/')

for f in files:
	if re.match('^frontend/controlcenter/.*\.(vue|js)$', f) == None:
		continue
	
	f = re.sub('^frontend/controlcenter/', './', f)
	
	retval += run_and_check(["./node_modules/.bin/eslint","--fix",f])

os.chdir('../../')


print("\n##### go files #####")

for f in files:
	if re.match('^.*\.go$', f) == None:
		continue
	
	retval += run_and_check(["gofmt","-l","-w","-s",f])
	retval += run_and_check(["golangci-lint","run",f])


print("\n##### end checks #####\n\n")

sys.exit(retval)
