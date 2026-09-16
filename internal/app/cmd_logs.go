package app

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"example.com/msh/internal/config"
	"example.com/msh/internal/executor"
	"example.com/msh/internal/exitcode"
	"example.com/msh/internal/inventory"
	"example.com/msh/internal/output"
	"example.com/msh/internal/shellquote"
)

const logsUsage = `Usage: msh logs ([user@]host | --host LIST | --group NAME | --tag K=V | --all) --file PATH [--lines N] [--follow] [flags]

Shows the last lines of a log file on each selected server (tail -n N).
With --follow, new lines are streamed live, prefixed with the server
name, until you press CTRL+C. Paths must be absolute and inside one of
the allowed_log_paths directories (default /var/log).
`

// maxLogLines bounds --lines to keep output and memory reasonable.
const maxLogLines = 100000

func logsCommand() *command {
	return &command{
		name:    "logs",
		summary: "Show or follow a log file on servers",
		usage:   logsUsage,
		run:     runLogs,
	}
}

func runLogs(c *cli, args []string) exitcode.Code {
	var opts commonOptions
	var sel selectionOptions
	var ro remoteOptions
	fs := c.newFlagSet("logs", logsUsage)
	opts.register(fs)
	sel.register(fs)
	sel.registerWhere(fs)
	ro.register(fs)
	file := fs.String("file", "", "absolute `path` of the log file")
	lines := fs.Int("lines", 50, "number of `lines` to show (1-100000)")
	follow := fs.Bool("follow", false, "stream new lines until CTRL+C")
	if _, code, ok := parseRemote(fs, &sel, args, 0); !ok {
		return code
	}
	if *lines < 1 || *lines > maxLogLines {
		c.errorf("logs: --lines must be between 1 and %d", maxLogLines)
		return exitcode.Usage
	}
	st, code := c.load(&opts, false)
	if st == nil {
		return code
	}
	if err := validateLogPath(*file, st.settings.AllowedLogPaths); err != nil {
		c.errorf("logs: %v", err)
		return exitcode.Usage
	}

	words := []string{"tail", "-n", strconv.Itoa(*lines)}
	if *follow {
		// -F keeps following across log rotation.
		words = append(words, "-F")
	}
	words = append(words, "--", *file)
	run := remoteRun{
		opts: &opts, sel: &sel, ro: &ro, name: "logs",
		req: executor.Request{Command: shellquote.Join(words)},
	}
	if !*follow {
		return c.runRemote(run)
	}
	if opts.format != output.Text {
		c.errorf("logs: --follow streams text; it cannot be combined with --output %s", opts.format)
		return exitcode.Usage
	}
	// Following has no natural end: no command timeout unless the user
	// asked for one on the command line.
	if st.settings.Sources[config.KeyCommandTimeout] != config.SourceFlag {
		run.unlimited = true
	}
	return c.runStreaming(run)
}

// runStreaming runs r with live, line-prefixed output. The final results
// only report status; their output was already shown.
func (c *cli) runStreaming(r remoteRun) exitcode.Code {
	var mu sync.Mutex
	var writers []*output.LineWriter
	orig := c.stdout
	r.req.OutputLimit = 64 << 10 // keep only a little; it was streamed already
	r.req.Stream = func(s inventory.Server) (io.Writer, io.Writer) {
		mu.Lock()
		defer mu.Unlock()
		name := output.PadName(s.Name, r.ro.nameWidth)
		out := output.NewLineWriter(orig, name+": ", &mu)
		errw := output.NewLineWriter(orig, output.StderrName(s.Name, r.ro.nameWidth)+": ", &mu)
		writers = append(writers, out, errw)
		return out, errw
	}
	r.ro.prefix = true
	r.quietOutput = true
	// The live writers (SSH goroutines) and the result sink (collector
	// goroutine) both write to stdout: route the sink through the same
	// mutex. The line writers keep the original writer, because they
	// already hold the mutex while writing.
	c.stdout = output.NewLockedWriter(orig, &mu)
	defer func() { c.stdout = orig }()
	code := c.runRemote(r)
	for _, w := range writers {
		w.Flush()
	}
	return code
}

// validateLogPath accepts only clean absolute paths inside an allowed
// directory. Symbolic links on the server are not resolved by msh.
func validateLogPath(path string, allowed []string) error {
	switch {
	case path == "":
		return errors.New("--file is required")
	case strings.ContainsAny(path, "\x00\n"):
		return errors.New("the path contains control characters")
	case !filepath.IsAbs(path):
		return fmt.Errorf("%q is not an absolute path", path)
	case filepath.Clean(path) != path:
		return fmt.Errorf("%q is not a clean path (use %q)", path, filepath.Clean(path))
	}
	for _, dir := range allowed {
		dir = strings.TrimSuffix(filepath.Clean(dir), "/")
		if dir != "" && strings.HasPrefix(path, dir+"/") {
			return nil
		}
	}
	return fmt.Errorf("%q is outside allowed_log_paths (%s)", path, strings.Join(allowed, ", "))
}
