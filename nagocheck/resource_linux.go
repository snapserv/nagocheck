/*
 * nagocheck - Reliable and lightweight Nagios plugins written in Go
 * Copyright (C) 2018-2019  Pascal Mathis
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package nagocheck

import (
	"os"
	"syscall"
)

const shmOpenFlags = os.O_CREATE | syscall.O_DSYNC | syscall.O_RSYNC
const shmReadFlags = shmOpenFlags | os.O_RDONLY
const shmWriteFlags = shmOpenFlags | os.O_WRONLY | os.O_TRUNC
const shmDefaultMode = 0600
