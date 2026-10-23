package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type apiOptions struct {
	command, cwd, forwardTo, prefix string
}

const apiStartupTimeout = 10 * time.Second
const apiShutdownTimeout = 3 * time.Second

func registerAPIFlags(flags *flag.FlagSet, api *apiOptions) {
	flags.StringVar(&api.command, "api-command", "", "backend executable and arguments; no shell")
	flags.StringVar(&api.cwd, "api-cwd", "", "backend working directory; defaults to current directory")
	flags.StringVar(&api.forwardTo, "api-forward-to", "", "backend HTTP(S) URL to forward API requests to")
	flags.StringVar(&api.prefix, "api-prefix", "/api", "request path prefix to forward, preserved in backend requests")
}

// Commands are executable/argument lists, never shell scripts. Preserve literal
// backslashes so Windows paths need no extra escaping. Either quote style groups
// spaces; use the other style when an argument needs a literal quote.
func splitAPICommand(command string) ([]string, error) {
	var args []string
	var word strings.Builder
	var quote rune
	started := false
	for _, ch := range command {
		switch {
		case quote != 0:
			if ch == quote {
				quote = 0
			} else {
				word.WriteRune(ch)
			}
		case ch == '\'' || ch == '"':
			quote = ch
			started = true
		case unicode.IsSpace(ch):
			if started {
				args = append(args, word.String())
				word.Reset()
				started = false
			}
		default:
			word.WriteRune(ch)
			started = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("api-command has an unclosed quote")
	}
	if started {
		args = append(args, word.String())
	}
	if len(args) == 0 || args[0] == "" {
		return nil, fmt.Errorf("api-command must name an executable")
	}
	return args, nil
}

func apiHandler(root http.Handler, options apiOptions) (http.Handler, *url.URL, error) {
	if options.cwd != "" && options.command == "" {
		return nil, nil, fmt.Errorf("api-cwd requires api-command")
	}
	if options.forwardTo == "" {
		return root, nil, nil
	}
	target, err := url.Parse(options.forwardTo)
	if err != nil || target == nil || target.Hostname() == "" ||
		(target.Scheme != "http" && target.Scheme != "https") ||
		target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return nil, nil, fmt.Errorf("api-forward-to must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	if port := target.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > maxPort {
			return nil, nil, fmt.Errorf("api-forward-to port must be between 1 and %d", maxPort)
		}
	}
	prefix := strings.TrimRight(options.prefix, "/")
	if options.prefix == "" || !strings.HasPrefix(options.prefix, "/") || strings.ContainsAny(options.prefix, "?#") {
		return nil, nil, fmt.Errorf("api-prefix must be a URL path starting with /")
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(r *http.Request) {
		// The local proxy supplies these headers itself instead of trusting values
		// sent by the browser. Keep the original host available to backend apps.
		host := r.Host
		proto := "http"
		if r.TLS != nil {
			proto = "https"
		}
		r.Header.Del("Forwarded")
		r.Header.Del("X-Forwarded-For")
		r.Header.Del("X-Forwarded-Host")
		r.Header.Del("X-Forwarded-Proto")
		director(r)
		r.Header.Set("X-Forwarded-Host", host)
		r.Header.Set("X-Forwarded-Proto", proto)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Match path segments: /api and /api/... belong to the API; /apiculture
		// remains a static file. Forward the complete path, including the prefix.
		if prefix == "" || r.URL.Path == prefix || strings.HasPrefix(r.URL.Path, prefix+"/") {
			proxy.ServeHTTP(w, r)
		} else {
			root.ServeHTTP(w, r)
		}
	}), target, nil
}

type apiProcess struct {
	cmd     *exec.Cmd
	tree    *processTree
	done    chan struct{}
	waitErr error // Read only after done closes.
}

func startAPI(options apiOptions) (*apiProcess, error) {
	args, err := splitAPICommand(options.command)
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(args[0]) && strings.ContainsAny(args[0], `/\`) {
		cwd := options.cwd
		if cwd == "" {
			cwd = "."
		}
		// Resolve explicit executable paths before exec.Command's Windows lookup;
		// otherwise it searches the parent's directory instead of API_CWD.
		args[0], err = filepath.Abs(filepath.Join(cwd, args[0]))
		if err != nil {
			return nil, err
		}
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = options.cwd
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	tree, err := newProcessTree(cmd)
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		tree.close()
		return nil, fmt.Errorf("start API: %v", err)
	}
	if err := tree.attach(cmd); err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		tree.close()
		return nil, fmt.Errorf("manage API process: %v", err)
	}
	process := &apiProcess{cmd: cmd, tree: tree, done: make(chan struct{})}
	go func() {
		process.waitErr = cmd.Wait()
		close(process.done)
	}()
	return process, nil
}

func (p *apiProcess) exited() error {
	if p.waitErr == nil {
		return fmt.Errorf("API exited unexpectedly with status 0")
	}
	return fmt.Errorf("API exited unexpectedly: %v", p.waitErr)
}

func (p *apiProcess) stop() error {
	defer p.tree.close()
	select {
	case <-p.done:
	default:
		// First let the backend handle an interrupt. Always kill the remaining
		// tree afterwards: a wrapper may exit before its children finish.
		p.tree.interrupt()
		timer := time.NewTimer(apiShutdownTimeout)
		defer timer.Stop()
		select {
		case <-p.done:
		case <-timer.C:
		}
	}
	if err := p.tree.kill(); err != nil {
		return fmt.Errorf("stop API process tree: %v", err)
	}
	<-p.done
	return nil
}

func apiAddress(target *url.URL) string {
	port := target.Port()
	if port == "" {
		port = "80"
		if target.Scheme == "https" {
			port = "443"
		}
	}
	return net.JoinHostPort(target.Hostname(), port)
}

func waitForAPI(ctx context.Context, process *apiProcess, target *url.URL) error {
	startup, cancel := context.WithTimeout(ctx, apiStartupTimeout)
	defer cancel()
	for {
		select {
		case <-process.done:
			return process.exited()
		case <-startup.Done():
			return fmt.Errorf("wait for API at %s: %v", target, startup.Err())
		default:
		}
		connection, err := (&net.Dialer{Timeout: 100 * time.Millisecond}).DialContext(startup, "tcp", apiAddress(target))
		if err == nil {
			connection.Close()
			return nil
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-process.done:
			timer.Stop()
			return process.exited()
		case <-startup.Done():
			timer.Stop()
			return fmt.Errorf("wait for API at %s: %v", target, startup.Err())
		case <-timer.C:
		}
	}
}
