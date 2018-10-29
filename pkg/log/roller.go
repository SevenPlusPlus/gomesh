/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"io"
	"path/filepath"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Roller implements a type that provides a rolling logger.
type Roller struct {
	Filename   string
	MaxSize    int
	MaxAge     int
	MaxBackups int
	Compress   bool
	LocalTime  bool
}

// GetLogWriter returns an io.Writer that writes to a rolling logger.
// This should be called only from the main goroutine (like during
// server setup) because this method is not thread-safe; it is careful
// to create only one log writer per log file, even if the log file
// is shared by different sites or middlewares. This ensures that
// rolling is synchronized, since a process (or multiple processes)
// should not create more than one roller on the same file at the
// same time. See issue #1363.
func (l Roller) GetLogWriter() io.Writer {
	absPath, err := filepath.Abs(l.Filename)
	if err != nil {
		absPath = l.Filename // oh well, hopefully they're consistent in how they specify the filename
	}
	lj, has := lumberjacks[absPath]
	if !has {
		lj = &lumberjack.Logger{
			Filename:   l.Filename,
			MaxSize:    l.MaxSize,
			MaxAge:     l.MaxAge,
			MaxBackups: l.MaxBackups,
			Compress:   l.Compress,
			LocalTime:  l.LocalTime,
		}
		lumberjacks[absPath] = lj
	}
	return lj
}


// DefaultRoller will roll logs by default.
func DefaultRoller() *Roller {
	return &Roller{
		MaxSize:    defaultRotateSize,
		MaxAge:     defaultRotateAge,
		MaxBackups: defaultRotateKeep,
		Compress:   false,
		LocalTime:  true,
	}
}

const (
	// defaultRotateSize is 100 MB.
	defaultRotateSize = 100
	// defaultRotateAge is 14 days.
	defaultRotateAge = 14
	// defaultRotateKeep is 10 files.
	defaultRotateKeep = 10
)

// lumberjacks maps log filenames to the logger
// that is being used to keep them rolled/maintained.
var lumberjacks = make(map[string]*lumberjack.Logger)
