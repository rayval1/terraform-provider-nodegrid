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

	// JumpHost, when set, is an intermediate device to tunnel through — the
	// equivalent of ssh -J. Nodegrid console servers routinely sit on a NAT'd
	// LAN behind a router unit, reachable only from that unit. The jump host
	// is reached with the same credentials and port as the target.
	JumpHost string
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
	addr := c.addr(c.cfg.Host)
	conn, cleanup, err := c.dial()
	if err != nil {
		return "", err
	}
	defer cleanup()

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

// sshConfig builds the client config used for both the target and, when
// tunnelling, the jump host.
func (c *Client) sshConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
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
}

func (c *Client) addr(host string) string {
	return net.JoinHostPort(host, fmt.Sprintf("%d", c.cfg.Port))
}

// dial connects to the target device, tunnelling through JumpHost when one is
// configured. The returned cleanup closes the tunnel and, when used, the
// bastion connection underneath it.
func (c *Client) dial() (*ssh.Client, func(), error) {
	sshCfg := c.sshConfig()
	target := c.addr(c.cfg.Host)

	if c.cfg.JumpHost == "" {
		conn, err := ssh.Dial("tcp", target, sshCfg)
		if err != nil {
			return nil, nil, fmt.Errorf("ssh dial %s: %w", target, err)
		}
		return conn, func() { conn.Close() }, nil
	}

	jump := c.addr(c.cfg.JumpHost)
	bastion, err := ssh.Dial("tcp", jump, sshCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("ssh dial jump host %s: %w", jump, err)
	}

	// Open a TCP connection to the target from the bastion, then run a second
	// SSH handshake over it, so authentication to the target is end-to-end
	// rather than delegated to the jump host.
	tunnel, err := bastion.Dial("tcp", target)
	if err != nil {
		bastion.Close()
		return nil, nil, fmt.Errorf("dial %s via jump host %s: %w", target, jump, err)
	}

	ncc, chans, reqs, err := ssh.NewClientConn(tunnel, target, sshCfg)
	if err != nil {
		tunnel.Close()
		bastion.Close()
		return nil, nil, fmt.Errorf("ssh handshake with %s via jump host %s: %w", target, jump, err)
	}

	conn := ssh.NewClient(ncc, chans, reqs)
	return conn, func() {
		conn.Close()
		bastion.Close()
	}, nil
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

// ParseExport extracts settings from export_settings output, ignoring
// prompts, echoes, and blank lines.
//
// Nodegrid emits one setting per line as "<section> <field>=<value>", with a
// SPACE between the section path and the field name:
//
//	/settings/network_settings hostname=in-bang-dc-1-105-cc1
//	/settings/network_settings global_dns_servers="10.0.0.1 10.0.0.2"
//
// Those are normalized to the slash-joined form this provider uses everywhere
// else ("/settings/network_settings/hostname"), so the keys returned here
// match the keys callers write in a `settings` map. Lines already in the
// slash-joined form are accepted unchanged.
func ParseExport(out string) map[string]string {
	settings := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "/") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		value := unquoteValue(strings.TrimSpace(line[eq+1:]))

		// "<section> <field>" -> "<section>/<field>". Split on the FIRST
		// whitespace: neither a section path nor a field name contains any, so
		// a residual space means the line is CLI chatter, not a setting.
		if i := strings.IndexAny(key, " \t"); i >= 0 {
			section := key[:i]
			field := strings.TrimSpace(key[i+1:])
			if section == "" || field == "" || strings.ContainsAny(field, " \t") {
				continue
			}
			key = section + "/" + field
		}
		settings[key] = value
	}
	return settings
}

// unquoteValue reverses quoteValue for values the CLI echoes back quoted,
// e.g. global_dns_servers="10.0.0.1 10.0.0.2". Without this, a value written
// as an unquoted string would read back with quotes and report drift forever.
func unquoteValue(v string) string {
	if len(v) < 2 || !strings.HasPrefix(v, `"`) || !strings.HasSuffix(v, `"`) {
		return v
	}
	inner := v[1 : len(v)-1]
	inner = strings.ReplaceAll(inner, `\"`, `"`)
	inner = strings.ReplaceAll(inner, `\\`, `\`)
	return inner
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
