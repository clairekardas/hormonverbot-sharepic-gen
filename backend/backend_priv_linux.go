// SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"strings"

	syscallset "github.com/oxzi/syscallset-go"
)

// This file reduces the backend's privileges by limiting the syscalls to a
// previously defined subset. This is enforced through seccomp-bpf.

// ToLeastPrivilege uses seccomp-bpf to limit the system calls.
//
// To inspect and debug the used syscalls, change LimitTo to LimitAndLog,
// startup auditd, and run the backend program:
//
//   $ sudo auditd -f | tee audit.log
//   $ ./backend
//
func ToLeastPrivilege() error {
	if !syscallset.IsSupported() {
		return fmt.Errorf("seccomp-bpf or syscallset is not supported")
	}

	filter := []string{
		"@basic-io",
		"@file-system",
		"@io-event",
		"@ipc",
		"@network-io",
		"@process ~execve ~execveat ~fork ~kill",
		"@signal",
		"fadvise64 ioctl madvise mremap sysinfo",
	}
	return syscallset.LimitTo(strings.Join(filter, " "))
}
