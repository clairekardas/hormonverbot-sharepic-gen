// SPDX-FileCopyrightText: Free Software Foundation Europe <https://fsfe.org>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !linux

package main

import (
	"fmt"
)

// This file should encourage developers to drop the backend's privileges with
// a call to the platform specific ToLeastPrivilege() function.
// On a Linux, this function might enforce a seccomp policy. On an OpenBSD,
// there might be a pledge. On a Windows, there might be… something?

// ToLeastPrivilege is not possible for an unsupported platform.
func ToLeastPrivilege() error {
	return fmt.Errorf("no implementation for this platform")
}
