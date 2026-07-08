// Package client drives the Nodegrid CLI over SSH.
//
// Nodegrid exposes its whole configuration as a tree of
// /settings/<section>/<field>=<value> pairs. Reads use `export_settings`,
// which prints those pairs; writes use `cd <section>` + `set field=value` +
// `commit`, the same commands an admin would type interactively. This is the
// same interface ZPE's own Ansible collection automates — the on-device REST
// API is not required or enabled here.
package client

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Timeout  time.Duration
}

type Client struct {
	cfg Config
}

func New(cfg Config) *Client {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{cfg: cfg}
}

// Run opens one SSH session to the device CLI, feeds it the given commands on
// stdin (exactly like the `ssh host <<EOF` heredocs this replaces), and
// returns the combined output.
func (c *Client) Run(commands []string) (string, error) {
	addr := net.JoinHostPort(c.cfg.Host, fmt.Sprintf("%d", c.cfg.Port))
	sshCfg := &ssh.ClientConfig{
		User: c.cfg.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.cfg.Password),
			ssh.KeyboardInteractive(func(_, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = c.cfg.Password
				}
				return answers, nil
			}),
		},
		// Console servers get reimaged and re-keyed; pinning host keys here
		// would just recreate the StrictHostKeyChecking=no behavior debate.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.cfg.Timeout,
	}

	conn, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return "", fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh session %s: %w", addr, err)
	}
	defer session.Close()

	var out strings.Builder
	session.Stdout = &out
	session.Stderr = &out

	script := strings.Join(commands, "\n") + "\nexit\n"
	session.Stdin = strings.NewReader(script)

	if err := session.Shell(); err != nil {
		return "", fmt.Errorf("ssh shell %s: %w", addr, err)
	}

	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	select {
	case err := <-done:
		// The Nodegrid CLI exits non-zero on some benign paths; command
		// failures are detected from the output text instead.
		_ = err
	case <-time.After(c.cfg.Timeout + 30*time.Second):
		return out.String(), fmt.Errorf("ssh session to %s timed out", addr)
	}

	return out.String(), nil
}

// RunChecked runs the commands and returns an error if the CLI reported one.
func (c *Client) RunChecked(commands []string) (string, error) {
	out, err := c.Run(commands)
	if err != nil {
		return out, err
	}
	if cliErr := findCLIError(out); cliErr != "" {
		return out, fmt.Errorf("nodegrid CLI error on %s: %s", c.cfg.Host, cliErr)
	}
	return out, nil
}

// GetSettings runs export_settings for each path prefix and returns every
// /settings/...=value pair found, keyed by full path.
func (c *Client) GetSettings(prefixes []string) (map[string]string, error) {
	cmds := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		cmds = append(cmds, "export_settings "+p)
	}
	out, err := c.Run(cmds)
	if err != nil {
		return nil, err
	}
	if cliErr := findCLIError(out); cliErr != "" {
		return nil, fmt.Errorf("nodegrid CLI error on %s: %s", c.cfg.Host, cliErr)
	}
	return ParseExport(out), nil
}

// ApplySettings writes the given full-path=value pairs and commits once.
// Paths sharing a parent section are grouped into a single cd + set batch.
func (c *Client) ApplySettings(settings map[string]string) error {
	if len(settings) == 0 {
		return nil
	}

	bySection := map[string]map[string]string{}
	for path, value := range settings {
		section, field, err := SplitPath(path)
		if err != nil {
			return err
		}
		if bySection[section] == nil {
			bySection[section] = map[string]string{}
		}
		bySection[section][field] = value
	}

	sections := make([]string, 0, len(bySection))
	for s := range bySection {
		sections = append(sections, s)
	}
	sort.Strings(sections)

	var cmds []string
	for _, section := range sections {
		cmds = append(cmds, "cd "+section)
		fields := bySection[section]
		names := make([]string, 0, len(fields))
		for f := range fields {
			names = append(names, f)
		}
		sort.Strings(names)
		for _, f := range names {
			cmds = append(cmds, fmt.Sprintf("set %s=%s", f, quoteValue(fields[f])))
		}
	}
	cmds = append(cmds, "commit")

	out, err := c.Run(cmds)
	if err != nil {
		return err
	}
	if cliErr := findCLIError(out); cliErr != "" {
		return fmt.Errorf("nodegrid CLI error on %s: %s", c.cfg.Host, cliErr)
	}
	return nil
}

// SplitPath turns "/settings/network_settings/hostname" into its section
// ("/settings/network_settings") and field ("hostname").
func SplitPath(path string) (section, field string, err error) {
	if !strings.HasPrefix(path, "/") {
		return "", "", fmt.Errorf("setting path %q must start with /", path)
	}
	idx := strings.LastIndex(path, "/")
	if idx <= 0 || idx == len(path)-1 {
		return "", "", fmt.Errorf("setting path %q must be /<section...>/<field>", path)
	}
	return path[:idx], path[idx+1:], nil
}

// ParseExport extracts path=value lines from export_settings output,
// ignoring prompts, echoes, and blank lines.
func ParseExport(out string) map[string]string {
	settings := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(strings.TrimSpace(line), "\r")
		if !strings.HasPrefix(line, "/") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if strings.ContainsAny(key, " \t") {
			continue // not a settings path (e.g. CLI chatter)
		}
		settings[key] = strings.TrimSpace(line[eq+1:])
	}
	return settings
}

func quoteValue(v string) string {
	escaped := strings.ReplaceAll(v, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func findCLIError(out string) string {
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "error") || strings.Contains(lower, "invalid value") {
			return trimmed
		}
	}
	return ""
}
