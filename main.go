package main

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

type writer struct {
	buf        []byte
	w          io.Writer
	redactions [][]byte
}

func (w *writer) Write(p []byte) (n int, err error) {
	// Append p to the buffer
	w.buf = append(w.buf, p...)

	// Send anything with a new line to the underlying writer
	for {
		idx := strings.IndexByte(string(w.buf), '\n')
		if idx == -1 {
			break
		}
		line := w.buf[:idx+1]
		w.buf = w.buf[idx+1:]

		// Redact any occurrences of the redactions in the line
		redactedLine := line
		for _, r := range w.redactions {
			redactedLine = []byte(strings.ReplaceAll(string(redactedLine), string(r), strings.Repeat("*", len(r))))
		}

		// Write the redacted line to the underlying writer
		_, err := w.w.Write(redactedLine)
		if err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func main() {
	// Get the first argument from the command line
	if len(os.Args) < 3 {
		_, _ = os.Stderr.WriteString("Not enough arguments: <new line split redactions> <cmd> [...args]\n")
		return
	}
	firstArg := os.Args[1]
	cmd := os.Args[2]
	args := os.Args[3:]

	// Split the first argument by new lines to get redactions
	redactions := strings.Split(firstArg, "\n")
	if len(redactions) == 0 {
		_, _ = os.Stderr.WriteString("No redactions provided\n")
		return
	}

	// Convert redactions to [][]byte
	redactionBytes := make([][]byte, len(redactions))
	for i, r := range redactions {
		redactionBytes[i] = []byte(r)
	}

	// Create the command
	c := exec.Command(cmd, args...)
	c.Env = os.Environ()

	// Start the command with a PTY
	ptmx, err := pty.Start(c)
	if err != nil {
		_, _ = os.Stderr.WriteString("Failed to start PTY: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer ptmx.Close()

	// Handle PTY size changes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize
	defer func() { signal.Stop(ch); close(ch) }()

	// Set stdin to raw mode if it's a terminal
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			defer term.Restore(int(os.Stdin.Fd()), oldState)
		}
	}

	// Copy stdin to PTY
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	// Copy PTY output through redacting writer to stdout
	redactingWriter := &writer{w: os.Stdout, redactions: redactionBytes}
	_, _ = io.Copy(redactingWriter, ptmx)

	// Wait for the command to finish
	if err := c.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
