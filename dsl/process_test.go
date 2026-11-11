/*
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package dsl

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestReadProcessLinesShortAndLong(t *testing.T) {
	long := strings.Repeat("x", bufio.MaxScanTokenSize+128)
	input := "short\n" + long + "\nmiddle\n" + long + "\n"

	var got []string
	err := readProcessLines(bytes.NewBufferString(input), func(line string) error {
		got = append(got, line)
		return nil
	})
	if err != nil {
		t.Fatalf("readProcessLines: %v", err)
	}
	want := []string{"short", long, "middle", long}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d mismatch: got len=%d want len=%d", i, len(got[i]), len(want[i]))
		}
	}
}

func TestReadProcessLinesNoTrailingNewline(t *testing.T) {
	long := strings.Repeat("y", bufio.MaxScanTokenSize+16)
	var got []string
	err := readProcessLines(bytes.NewBufferString(long), func(line string) error {
		got = append(got, line)
		return nil
	})
	if err != nil {
		t.Fatalf("readProcessLines: %v", err)
	}
	if len(got) != 1 || got[0] != long {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestTrimEOL(t *testing.T) {
	if TrimEOL("hi\n") != "hi" {
		t.Fatal("expected single newline trimmed")
	}
	if TrimEOL("hi\r\n") != "hi" {
		t.Fatal("expected CRLF trimmed")
	}
	if TrimEOL("hi\n\n") != "hi\n" {
		t.Fatal("expected only one newline trimmed")
	}
}

func TestProcessEmitsLongStdoutLine(t *testing.T) {
	long := strings.Repeat("z", bufio.MaxScanTokenSize+64)
	ctx := NewCtx(nil)
	p := &Process{
		Name:    "long-stdout",
		Command: "bash",
		Args: []string{
			"-c",
			fmt.Sprintf("python3 -c \"import sys; sys.stdout.write('z'*%d + chr(10)); sys.stdout.flush()\"", len(long)),
		},
	}
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = p.Kill(ctx) }()

	select {
	case line := <-p.Stdout:
		if line != long {
			t.Fatalf("got len=%d want len=%d", len(line), len(long))
		}
	case errLine := <-p.Stderr:
		t.Fatalf("unexpected stderr: %s", errLine)
	case code := <-p.ExitCode:
		t.Fatalf("process exited early with %d before stdout", code)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for long stdout line")
	}
}
