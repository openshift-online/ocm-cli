/*
Copyright (c) 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package output

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/mattn/go-isatty"
)

// PrinterBuilder contains the data and logic needed to create new printers.
type PrinterBuilder struct {
	writer io.Writer
	pager  string
}

// Printer knows how to write output text.
type Printer struct {
	writer      io.Writer
	pagerCmd    *exec.Cmd
	pagerStop   chan int
	pagerReader *os.File
	pagerWriter *os.File
}

// Make sure that we implement the io.Writer interface.
var _ io.Writer = (*Printer)(nil)

// NewPrinter creates a builder that can then be used to configure and create a printer.
func NewPrinter() *PrinterBuilder {
	return &PrinterBuilder{
		writer: os.Stdout,
	}
}

// Writer sets the writer where the printer will write. It will usually be a file or the standard
// output fo the process. This is mandatory.
func (b *PrinterBuilder) Writer(value io.Writer) *PrinterBuilder {
	b.writer = value
	return b
}

// Pager indicates the command that will be used to display output page by page. If empty no pager
// will be used.
func (b *PrinterBuilder) Pager(value string) *PrinterBuilder {
	b.pager = value
	return b
}

// Build uses the data stored in the builder to create a new printer.
func (b *PrinterBuilder) Build(ctx context.Context) (result *Printer, err error) {
	// Check parameters:
	if b.writer == nil {
		err = fmt.Errorf("writer is mandatory")
		return
	}

	// Check if there pager tool is available:
	pagerPath, err := b.pagerPath()
	if err != nil {
		return
	}

	// Check if the output is a TTY:
	isTTY, err := b.isTTY(b.writer)
	if err != nil {
		return
	}

	// If paging is enabled, a pager is available and the output is a TTY, then start that pager
	// in the background and redirect all the output to it:
	writer := b.writer
	var pagerCmd *exec.Cmd
	var pagerStop chan int
	var pagerReader, pagerWriter *os.File
	if pagerPath != "" && isTTY {
		// Create a pipe to connect us to the pager process:
		pagerReader, pagerWriter, err = os.Pipe()
		if err != nil {
			return
		}

		// Start the pager process so that it reads from the pipe and writes to our output:
		pagerCmd = exec.Command(pagerPath)
		pagerCmd.Stdin = pagerReader
		pagerCmd.Stdout = writer
		err = pagerCmd.Start()
		if err != nil {
			pagerReader.Close()
			pagerWriter.Close()
			return
		}

		// The pager process may finish at any time, even before we finish writing, because
		// the user may explicitly finish, with the `q` command or with Ctr-C. That means
		// that we need to wait for the process to finish in a separate goroutine. When it
		// finishes we then need to close both ends of the pipe. That will result in
		// returning an error to any goroutine that tries to write to it, and that will in
		// turn result in gracefully ending that goroutine.
		pagerStop = make(chan int)
		go func() {
			pagerCmd.Wait()
			pagerReader.Close()
			pagerWriter.Close()
			close(pagerStop)
		}()
	}

	// Create and populate the object:
	result = &Printer{
		writer:      writer,
		pagerCmd:    pagerCmd,
		pagerStop:   pagerStop,
		pagerReader: pagerReader,
		pagerWriter: pagerWriter,
	}

	return
}

// isTTY checks if the given writer is a TTY.
func (b *PrinterBuilder) isTTY(writer io.Writer) (result bool, err error) {
	file, ok := writer.(*os.File)
	if ok {
		result = isatty.IsTerminal(file.Fd())
	}
	return
}

// pagerPath checks if the `less` command line tool is available and retursn its path.
func (b *PrinterBuilder) pagerPath() (result string, err error) {
	if b.pager == "" {
		return
	}
	result, err = exec.LookPath(b.pager)
	if errors.Is(err, exec.ErrNotFound) {
		err = nil
	}
	return
}

// Write is the implementation of the io.Writer interface.
func (p *Printer) Write(b []byte) (n int, err error) {
	writer := p.writer
	if p.pagerWriter != nil {
		writer = p.pagerWriter
	}
	n, err = writer.Write(b)
	return
}

// Close releases all the resources used by the printer.
func (p *Printer) Close() error {
	// At this point we assume that we finished writing. But the pager may still be running. To
	// make sure that it stops we need to close both ends of the pipe. Then we need to wait till
	// the goroutine that is responsible for waiting the process has finished, as otherwise we
	// may leave a zombie process around.
	if p.pagerCmd != nil {
		p.pagerReader.Close()
		p.pagerWriter.Close()
		<-p.pagerStop
	}
	return nil
}
