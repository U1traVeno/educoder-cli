package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	dataBaseURL       = "https://data.educoder.net"
	wwwBaseURL        = "https://www.educoder.net"
	defaultBundlePath = "/private/tmp/educoder-umi.js"
)

var stdinReader = bufio.NewReader(os.Stdin)

type config struct {
	courseID       string
	courseCode     string
	bundlePath     string
	credentialFile string
	zzud           string
	jsonOut        bool
}

type client struct {
	cfg       config
	http      *http.Client
	session   string
	autologin string
	ak        string
	sk        string
	login     string
}

type storedCredentials struct {
	Session   string `json:"session"`
	Autologin string `json:"autologin,omitempty"`
	Login     string `json:"login,omitempty"`
	Username  string `json:"username,omitempty"`
	Source    string `json:"source,omitempty"`
	SavedAt   string `json:"saved_at"`
}

type accountLoginResponse struct {
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	Login   string `json:"login,omitempty"`
	Name    string `json:"name,omitempty"`
	Cookies struct {
		Action string `json:"action,omitempty"`
		Value  string `json:"value,omitempty"`
	} `json:"cookies,omitempty"`
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	if !strings.Contains(value, "=") {
		return fmt.Errorf("expected key=value, got %q", value)
	}
	*m = append(*m, value)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, args, err := parseGlobalFlags(os.Args[1:])
	if err != nil {
		return err
	}
	if len(args) == 0 {
		usage()
		return errors.New("missing command")
	}

	command := args[0]
	if command == "logout" {
		return logout(cfg)
	}

	if command == "login" {
		fs := flag.NewFlagSet("login", flag.ExitOnError)
		username := fs.String("username", "", "Educoder username, phone, or email")
		passwordStdin := fs.Bool("password-stdin", false, "read password from stdin")
		_ = fs.Parse(args[1:])
		c, err := newClient(cfg, false)
		if err != nil {
			return err
		}
		return c.loginWithPassword(*username, *passwordStdin)
	}

	c, err := newClient(cfg, true)
	if err != nil {
		return err
	}

	switch command {
	case "whoami":
		return c.whoami()
	case "courses":
		return c.courses()
	case "labs":
		return c.labs()
	case "start":
		fs := flag.NewFlagSet("start", flag.ExitOnError)
		shixun := fs.String("shixun", "", "shixun identifier")
		_ = fs.Parse(args[1:])
		if *shixun == "" {
			return errors.New("start requires --shixun")
		}
		return c.start(*shixun)
	case "task":
		fs := flag.NewFlagSet("task", flag.ExitOnError)
		task := fs.String("task", "", "task/game identifier")
		_ = fs.Parse(args[1:])
		if *task == "" {
			return errors.New("task requires --task")
		}
		return c.task(*task)
	case "active-pod":
		fs := flag.NewFlagSet("active-pod", flag.ExitOnError)
		myshixun := fs.String("myshixun", "", "myshixun identifier")
		env := fs.Int("env", 0, "shixun_environment_id")
		tab := fs.Int("tab", 4, "tab_type")
		gameID := fs.Int("game-id", 0, "numeric game id")
		homeworkID := fs.String("homework-id", "", "homework_common_id")
		extras := multiFlag{}
		fs.Var(&extras, "extra", "additional query parameter as key=value; may be repeated")
		_ = fs.Parse(args[1:])
		if *myshixun == "" || *env == 0 || *gameID == 0 {
			return errors.New("active-pod requires --myshixun, --env, and --game-id")
		}
		return c.activePod(*myshixun, *env, *tab, *gameID, *homeworkID, extras)
	case "proxy-list":
		fs := flag.NewFlagSet("proxy-list", flag.ExitOnError)
		task := fs.String("task", "", "task/game identifier")
		_ = fs.Parse(args[1:])
		if *task == "" {
			return errors.New("proxy-list requires --task")
		}
		return c.proxyList(*task)
	case "port-proxy":
		fs := flag.NewFlagSet("port-proxy", flag.ExitOnError)
		task := fs.String("task", "", "task/game identifier")
		port := fs.Int("port", 0, "container port")
		_ = fs.Parse(args[1:])
		if *task == "" || *port <= 0 || *port > 65535 {
			return errors.New("port-proxy requires --task and a valid --port")
		}
		return c.portProxy(*task, *port)
	case "api-get":
		fs := flag.NewFlagSet("api-get", flag.ExitOnError)
		path := fs.String("path", "", "API path beginning with /api/")
		_ = fs.Parse(args[1:])
		if *path == "" {
			return errors.New("api-get requires --path")
		}
		return c.rawAPI(http.MethodGet, *path, "")
	case "api-post":
		fs := flag.NewFlagSet("api-post", flag.ExitOnError)
		path := fs.String("path", "", "API path beginning with /api/")
		body := fs.String("body", "{}", "JSON request body")
		_ = fs.Parse(args[1:])
		if *path == "" {
			return errors.New("api-post requires --path")
		}
		return c.rawAPI(http.MethodPost, *path, *body)
	case "vm-exec":
		fs := flag.NewFlagSet("vm-exec", flag.ExitOnError)
		myshixun := fs.String("myshixun", "", "myshixun identifier")
		env := fs.Int("env", 0, "shixun_environment_id")
		tab := fs.Int("tab", 4, "tab_type")
		gameID := fs.Int("game-id", 0, "numeric game id")
		homeworkID := fs.String("homework-id", "", "homework_common_id")
		command := fs.String("cmd", "pwd", "remote shell command")
		extras := multiFlag{}
		fs.Var(&extras, "extra", "additional query parameter as key=value; may be repeated")
		_ = fs.Parse(args[1:])
		if *myshixun == "" || *env == 0 || *gameID == 0 {
			return errors.New("vm-exec requires --myshixun, --env, and --game-id")
		}
		return c.vmExec(*myshixun, *env, *tab, *gameID, *homeworkID, extras, *command)
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func parseGlobalFlags(args []string) (config, []string, error) {
	cfg := config{}
	fs := flag.NewFlagSet("educoder", flag.ContinueOnError)
	fs.StringVar(&cfg.courseID, "course-id", "", "numeric Educoder course id; discovered when omitted")
	fs.StringVar(&cfg.courseCode, "course-code", "", "Educoder classroom code; optional course discovery hint")
	fs.StringVar(&cfg.bundlePath, "bundle", defaultBundlePath, "cached Educoder umi bundle")
	fs.StringVar(&cfg.credentialFile, "credentials", defaultCredentialFile(), "CLI credentials file")
	fs.StringVar(&cfg.zzud, "zzud", "", "Educoder login name; detected from whoami when omitted")
	fs.BoolVar(&cfg.jsonOut, "json", false, "print raw JSON responses")
	if err := fs.Parse(args); err != nil {
		return cfg, nil, err
	}
	return cfg, fs.Args(), nil
}

func usage() {
	_, _ = os.Stderr.WriteString(`Usage:
  educoder [global flags] whoami
  educoder [global flags] login --username <username>
  printf '%s\n' "$EDUCODER_PASSWORD" | educoder [global flags] login --username <username> --password-stdin
  educoder [global flags] logout
  educoder [global flags] courses
  educoder [global flags] labs
  educoder [global flags] start --shixun <shixun-id>
  educoder [global flags] task --task <game-id>
  educoder [global flags] active-pod --myshixun <myshixun-id> --env <env-id> --tab 4 --game-id <numeric-game-id>
  educoder [global flags] proxy-list --task <game-id>
  educoder [global flags] port-proxy --task <game-id> --port 22
  educoder [global flags] api-get --path /api/users/get_user_info.json
  educoder [global flags] api-post --path /api/tasks/<task>/proxy_list --body '{}'
  educoder [global flags] vm-exec --myshixun <myshixun-id> --env <env-id> --tab 4 --game-id <numeric-game-id> --homework-id <homework-id> --cmd 'pwd'

Global flags default to the OS course and a local CLI credentials file.
`)
}

func newClient(cfg config, needCredentials bool) (*client, error) {
	c := &client{cfg: cfg, http: &http.Client{Timeout: 30 * time.Second}}
	if needCredentials {
		if err := c.loadStoredCredentials(); err != nil {
			return nil, err
		}
	}
	if err := c.loadSigningKeys(); err != nil {
		return nil, err
	}
	return c, nil
}

func defaultCredentialFile() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil || home == "" {
			return "educoder-credentials.json"
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "educoder-helper", "credentials.json")
}

func (c *client) loadStoredCredentials() error {
	data, err := os.ReadFile(c.cfg.credentialFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("not logged in; run `educoder login --username <username>`, or pass --credentials")
		}
		return err
	}
	var creds storedCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return err
	}
	if creds.Session == "" {
		return fmt.Errorf("credentials file %s does not contain a session; run `educoder login` again", c.cfg.credentialFile)
	}
	c.session = creds.Session
	c.autologin = creds.Autologin
	c.login = creds.Login
	return nil
}

func (c *client) saveCredentials(login, username, source string) error {
	creds := storedCredentials{
		Session:   c.session,
		Autologin: c.autologin,
		Login:     login,
		Username:  username,
		Source:    source,
		SavedAt:   time.Now().Format(time.RFC3339),
	}
	if err := os.MkdirAll(filepath.Dir(c.cfg.credentialFile), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.cfg.credentialFile, append(data, '\n'), 0600)
}

func logout(cfg config) error {
	err := os.Remove(cfg.credentialFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("No Educoder credentials found at %s\n", cfg.credentialFile)
			return nil
		}
		return err
	}
	fmt.Printf("Removed Educoder credentials: %s\n", cfg.credentialFile)
	return nil
}

func (c *client) loadSigningKeys() error {
	bundle, err := os.ReadFile(c.cfg.bundlePath)
	if err != nil || len(bundle) == 0 {
		bundle, err = c.fetchBundle()
		if err != nil {
			return err
		}
	}
	re := regexp.MustCompile(`(?s)51459:function\([^)]*\).*?const t="([^"]+)",r="([^"]+)"`)
	m := re.FindSubmatch(bundle)
	if len(m) != 3 {
		return errors.New("could not find Educoder signing keys in umi bundle")
	}
	ak, err := doubleBase64Decode(string(m[1]))
	if err != nil {
		return err
	}
	sk, err := doubleBase64Decode(string(m[2]))
	if err != nil {
		return err
	}
	c.ak, c.sk = ak, sk
	return nil
}

func (c *client) fetchBundle() ([]byte, error) {
	pageURL := wwwBaseURL
	if c.cfg.courseCode != "" {
		pageURL = fmt.Sprintf("%s/classrooms/%s/shixun_homework", wwwBaseURL, c.cfg.courseCode)
	}
	req, _ := http.NewRequest(http.MethodGet, pageURL, nil)
	req.Header.Set("User-Agent", userAgent())
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	html, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`https://www-cdn\.educoder\.net/umi\.[^"']+\.js`)
	src := re.Find(html)
	if len(src) == 0 {
		return nil, errors.New("could not find umi bundle URL")
	}
	req, _ = http.NewRequest(http.MethodGet, string(src), nil)
	req.Header.Set("User-Agent", userAgent())
	resp, err = c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bundle, err := io.ReadAll(resp.Body)
	if err == nil && len(bundle) > 0 {
		_ = os.WriteFile(c.cfg.bundlePath, bundle, 0600)
	}
	return bundle, err
}

func doubleBase64Decode(s string) (string, error) {
	first, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	second, err := base64.StdEncoding.DecodeString(string(first))
	if err != nil {
		return "", err
	}
	return string(second), nil
}

func (c *client) newSignedRequest(method, apiPath string, body any, referer string) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(buf)
	}
	u := apiPath
	if strings.HasPrefix(apiPath, "/") {
		u = dataBaseURL + apiPath
	}
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	raw := fmt.Sprintf("method=%s&ak=%s&sk=%s&time=%s", method, c.ak, c.sk, ts)
	b64 := base64.StdEncoding.EncodeToString([]byte(raw))
	sum := md5.Sum([]byte(b64))
	sig := hex.EncodeToString(sum[:])

	req, err := http.NewRequest(method, u, reader)
	if err != nil {
		return nil, err
	}
	if referer == "" {
		referer = wwwBaseURL
		if c.cfg.courseCode != "" {
			referer = fmt.Sprintf("%s/classrooms/%s/shixun_homework", wwwBaseURL, c.cfg.courseCode)
		}
	}
	req.Header.Set("User-Agent", userAgent())
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", wwwBaseURL)
	req.Header.Set("Referer", referer)
	req.Header.Set("X-EDU-Type", "pc")
	req.Header.Set("X-EDU-Timestamp", ts)
	req.Header.Set("X-EDU-Signature", sig)
	req.Header.Set("X-Original-Protocol", "https:")
	req.Header.Set("X-Original-Host", "www.educoder.net")
	req.Header.Set("X-Original-Origin", wwwBaseURL)
	if c.session != "" {
		req.Header.Set("Pc-Authorization", c.session)
		req.Header.Set("Cookie", c.cookieHeader())
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	}
	return req, nil
}

func (c *client) signedRequest(method, apiPath string, body any, out any) error {
	req, err := c.newSignedRequest(method, apiPath, body, "")
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out == nil {
		fmt.Println(string(data))
		return nil
	}
	return json.Unmarshal(data, out)
}

func (c *client) loginWithPassword(username string, passwordStdin bool) error {
	username = strings.TrimSpace(username)
	if username == "" {
		var err error
		username, err = promptLine("Educoder username: ")
		if err != nil {
			return err
		}
		username = strings.TrimSpace(username)
	}
	if username == "" {
		return errors.New("login requires a username")
	}
	password, err := readPassword(passwordStdin)
	if err != nil {
		return err
	}
	if password == "" {
		return errors.New("login requires a password")
	}

	body := map[string]any{
		"login":     username,
		"password":  password,
		"autologin": true,
	}
	req, err := c.newSignedRequest(http.MethodPost, "/api/accounts/login.json", body, wwwBaseURL+"/login")
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("login HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result accountLoginResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	if result.Status != 0 && result.Message != "" {
		return fmt.Errorf("login failed: %s", result.Message)
	}
	c.session = strings.TrimSpace(resp.Header.Get("cs"))
	if c.session == "" {
		return errors.New("login response did not include an Educoder session header")
	}
	if result.Cookies.Action == "autologin" && result.Cookies.Value != "" {
		c.autologin = result.Cookies.Value
	}
	if c.autologin == c.session {
		c.autologin = ""
	}

	var info map[string]any
	if err := c.signedRequest(http.MethodGet, "/api/users/get_user_info.json", nil, &info); err != nil {
		return fmt.Errorf("login succeeded but session verification failed: %w", err)
	}
	login := firstNonEmpty(stringValue(info, "login"), result.Login)
	displayName := firstNonEmpty(stringValue(info, "username"), stringValue(info, "name"), result.Name)
	if err := c.saveCredentials(login, displayName, "password"); err != nil {
		return err
	}
	if c.cfg.jsonOut {
		return printJSON(map[string]any{
			"login":            login,
			"username":         displayName,
			"credentials_file": c.cfg.credentialFile,
		})
	}
	fmt.Printf("Educoder login OK: login=%s username=%s\nSaved credentials: %s\n", login, displayName, c.cfg.credentialFile)
	return nil
}

func promptLine(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	text, err := stdinReader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(text, "\r\n"), nil
}

func readPassword(fromStdin bool) (string, error) {
	if fromStdin {
		return promptLine("")
	}
	fmt.Fprint(os.Stderr, "Educoder password: ")
	if isTerminal(os.Stdin) {
		if err := setTerminalEcho(false); err != nil {
			return "", err
		}
		defer func() {
			_ = setTerminalEcho(true)
			fmt.Fprintln(os.Stderr)
		}()
	}
	return promptLine("")
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func setTerminalEcho(enabled bool) error {
	arg := "-echo"
	if enabled {
		arg = "echo"
	}
	cmd := exec.Command("stty", arg)
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (c *client) cookieHeader() string {
	parts := []string{"_educoder_session=" + c.session}
	if c.autologin != "" {
		parts = append(parts, "autologin_trustie="+c.autologin)
	}
	return strings.Join(parts, "; ")
}

func userAgent() string {
	return "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
}

func (c *client) ensureLogin() error {
	if c.login != "" {
		return nil
	}
	if c.cfg.zzud != "" {
		c.login = c.cfg.zzud
		return nil
	}
	var m map[string]any
	if err := c.signedRequest(http.MethodGet, "/api/users/get_user_info.json", nil, &m); err != nil {
		return err
	}
	c.login = stringValue(m, "login")
	if c.login == "" {
		return errors.New("could not detect Educoder login from get_user_info")
	}
	return nil
}

func (c *client) whoami() error {
	var m map[string]any
	if err := c.signedRequest(http.MethodGet, "/api/users/get_user_info.json", nil, &m); err != nil {
		return err
	}
	if c.cfg.jsonOut {
		return printJSON(m)
	}
	fmt.Printf("login=%s user_id=%s username=%s\n", stringValue(m, "login"), numberString(m, "user_id"), stringValue(m, "username"))
	return nil
}

func (c *client) loginStatus(persist bool) error {
	var m map[string]any
	if err := c.signedRequest(http.MethodGet, "/api/users/get_user_info.json", nil, &m); err != nil {
		return err
	}
	login := stringValue(m, "login")
	username := stringValue(m, "username")
	if persist {
		if err := c.saveCredentials(login, username, "session"); err != nil {
			return err
		}
	}
	if c.cfg.jsonOut {
		return printJSON(m)
	}
	if persist {
		fmt.Printf("Educoder login OK: login=%s username=%s\nSaved credentials: %s\n", login, username, c.cfg.credentialFile)
	} else {
		fmt.Printf("Educoder login OK: login=%s username=%s\n", login, username)
	}
	return nil
}

func (c *client) courses() error {
	items, err := c.fetchCourses()
	if err != nil {
		return err
	}
	if c.cfg.jsonOut {
		return printJSON(items)
	}
	for _, course := range items {
		fmt.Printf("%-8s code=%-10s homework=%-3s name=%s\n",
			numberString(course, "id"),
			emptyDash(courseCode(course)),
			numberString(course, "homework_commons_count"),
			stringValue(course, "name"),
		)
	}
	return nil
}

func (c *client) fetchCourses() ([]map[string]any, error) {
	if err := c.ensureLogin(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/api/users/%s/courses.json?page=1&per_page=100", url.PathEscape(c.login))
	var m map[string]any
	if err := c.signedRequest(http.MethodGet, path, nil, &m); err != nil {
		return nil, err
	}
	raw := arrayValue(m, "courses")
	items := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if course, ok := item.(map[string]any); ok {
			items = append(items, course)
		}
	}
	return items, nil
}

func (c *client) resolveCourseID() (string, error) {
	if c.cfg.courseID != "" {
		return c.cfg.courseID, nil
	}
	items, err := c.fetchCourses()
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", errors.New("no Educoder courses found for this account")
	}
	if c.cfg.courseCode != "" {
		for _, course := range items {
			if strings.EqualFold(courseCode(course), c.cfg.courseCode) {
				return numberString(course, "id"), nil
			}
		}
		return "", fmt.Errorf("could not find course with --course-code %q; run `educoder courses`", c.cfg.courseCode)
	}
	if len(items) == 1 {
		return numberString(items[0], "id"), nil
	}
	var osMatches []map[string]any
	for _, course := range items {
		name := strings.ToLower(stringValue(course, "name"))
		if strings.Contains(name, "操作系统") || strings.Contains(name, "operating system") {
			osMatches = append(osMatches, course)
		}
	}
	if len(osMatches) == 1 {
		return numberString(osMatches[0], "id"), nil
	}
	return "", errors.New("multiple courses found; run `educoder courses` and pass --course-id or --course-code")
}

func (c *client) labs() error {
	if err := c.ensureLogin(); err != nil {
		return err
	}
	courseID, err := c.resolveCourseID()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/courses/%s/homework_commons.json?homework_type=practice&page=1&limit=100&zzud=%s", courseID, url.QueryEscape(c.login))
	var m map[string]any
	if err := c.signedRequest(http.MethodGet, path, nil, &m); err != nil {
		return err
	}
	if c.cfg.jsonOut {
		return printJSON(m)
	}
	items, _ := m["homeworks"].([]any)
	for _, item := range items {
		hw, _ := item.(map[string]any)
		fmt.Printf("%-24s homework=%s shixun=%s myshixun=%s student_work=%s status=%s\n",
			stringValue(hw, "name"),
			numberString(hw, "homework_id"),
			stringValue(hw, "shixun_identifier"),
			emptyDash(stringValue(hw, "myshixun_identifier")),
			numberString(hw, "student_work_id"),
			statusString(hw["status"]),
		)
	}
	return nil
}

func (c *client) start(shixun string) error {
	if err := c.ensureLogin(); err != nil {
		return err
	}
	path := fmt.Sprintf("/api/shixuns/%s/shixun_exec.json?zzud=%s", url.PathEscape(shixun), url.QueryEscape(c.login))
	var m map[string]any
	if err := c.signedRequest(http.MethodGet, path, nil, &m); err != nil {
		return err
	}
	if c.cfg.jsonOut {
		return printJSON(m)
	}
	fmt.Printf("game_identifier=%s\n", stringValue(m, "game_identifier"))
	return nil
}

func (c *client) task(task string) error {
	if err := c.ensureLogin(); err != nil {
		return err
	}
	path := fmt.Sprintf("/api/tasks/%s.json?zzud=%s", url.PathEscape(task), url.QueryEscape(c.login))
	var m map[string]any
	if err := c.signedRequest(http.MethodGet, path, nil, &m); err != nil {
		return err
	}
	if c.cfg.jsonOut {
		return printJSON(m)
	}
	game, _ := m["game"].(map[string]any)
	my, _ := m["myshixun"].(map[string]any)
	shixun, _ := m["shixun"].(map[string]any)
	challenge, _ := m["challenge"].(map[string]any)
	fmt.Printf("task=%s game_id=%s status=%s\n", stringValue(game, "identifier"), numberString(game, "id"), numberString(game, "status"))
	fmt.Printf("shixun=%s name=%s myshixun=%s envs=%d\n", stringValue(shixun, "identifier"), stringValue(shixun, "name"), stringValue(my, "identifier"), len(arrayValue(m, "shixun_environments")))
	fmt.Printf("challenge=%s position=%s subject=%s\n", numberString(challenge, "id"), numberString(challenge, "position"), stringValue(challenge, "subject"))
	fmt.Printf("wss_url=%s git_url=%s\n", stringValue(m, "wss_url"), stringValue(m, "git_url"))
	return nil
}

func (c *client) activePod(myshixun string, env, tab, gameID int, homeworkID string, extras []string) error {
	if err := c.ensureLogin(); err != nil {
		return err
	}
	q := url.Values{}
	q.Set("shixun_environment_id", strconv.Itoa(env))
	q.Set("tab_type", strconv.Itoa(tab))
	q.Set("game_id", strconv.Itoa(gameID))
	q.Set("zzud", c.login)
	if homeworkID != "" {
		q.Set("homework_common_id", homeworkID)
	}
	for _, item := range extras {
		key, value, _ := strings.Cut(item, "=")
		q.Set(key, value)
	}
	path := fmt.Sprintf("/api/myshixuns/%s/active_pod.json?%s", url.PathEscape(myshixun), q.Encode())
	var m map[string]any
	if err := c.signedRequest(http.MethodGet, path, nil, &m); err != nil {
		return err
	}
	return printJSON(m)
}

func (c *client) proxyList(task string) error {
	if err := c.ensureLogin(); err != nil {
		return err
	}
	path := fmt.Sprintf("/api/tasks/%s/proxy_list?zzud=%s", url.PathEscape(task), url.QueryEscape(c.login))
	var m map[string]any
	if err := c.signedRequest(http.MethodPost, path, map[string]any{}, &m); err != nil {
		return err
	}
	return printJSON(m)
}

func (c *client) portProxy(task string, port int) error {
	if err := c.ensureLogin(); err != nil {
		return err
	}
	path := fmt.Sprintf("/api/tasks/%s/port_proxy?zzud=%s", url.PathEscape(task), url.QueryEscape(c.login))
	var m map[string]any
	if err := c.signedRequest(http.MethodPost, path, map[string]any{"port": port}, &m); err != nil {
		return err
	}
	return printJSON(m)
}

func (c *client) rawAPI(method, path, bodyText string) error {
	if !strings.HasPrefix(path, "/api/") {
		return errors.New("path must start with /api/")
	}
	var body any
	if method == http.MethodPost {
		if err := json.Unmarshal([]byte(bodyText), &body); err != nil {
			return err
		}
	}
	var m any
	if err := c.signedRequest(method, path, body, &m); err != nil {
		return err
	}
	return printJSON(m)
}

type podStartResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    struct {
		SSHAddress string `json:"ssh_address"`
		Port       string `json:"port"`
		Username   string `json:"username"`
		Password   string `json:"password"`
	} `json:"data"`
}

func (c *client) vmExec(myshixun string, env, tab, gameID int, homeworkID string, extras []string, remoteCommand string) error {
	if _, err := exec.LookPath("expect"); err != nil {
		return errors.New("vm-exec requires /usr/bin/expect or another expect executable")
	}
	pod, err := c.startPod(myshixun, env, tab, gameID, homeworkID, extras)
	if err != nil {
		return err
	}
	if pod.Data.SSHAddress == "" || pod.Data.Port == "" || pod.Data.Username == "" || pod.Data.Password == "" {
		return fmt.Errorf("start.json did not return SSH credentials: status=%d message=%s", pod.Status, pod.Message)
	}
	script := `set timeout 30
spawn ssh -tt -o PreferredAuthentications=password -o PubkeyAuthentication=no -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p $env(EDU_PORT) $env(EDU_USER)@$env(EDU_HOST)
expect "*assword:*"
send -- "$env(EDU_PASS)\r"
expect "*# "
send -- "$env(EDU_CMD)\r"
send -- "exit\r"
expect eof`
	cmd := exec.Command("expect", "-c", script)
	cmd.Env = append(os.Environ(),
		"EDU_HOST="+pod.Data.SSHAddress,
		"EDU_PORT="+pod.Data.Port,
		"EDU_USER="+pod.Data.Username,
		"EDU_PASS="+pod.Data.Password,
		"EDU_CMD="+remoteCommand,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (c *client) startPod(myshixun string, env, tab, gameID int, homeworkID string, extras []string) (*podStartResponse, error) {
	if err := c.ensureLogin(); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("shixun_environment_id", strconv.Itoa(env))
	q.Set("tab_type", strconv.Itoa(tab))
	q.Set("game_id", strconv.Itoa(gameID))
	q.Set("zzud", c.login)
	if homeworkID != "" {
		q.Set("homework_common_id", homeworkID)
	}
	for _, item := range extras {
		key, value, _ := strings.Cut(item, "=")
		q.Set(key, value)
	}
	path := fmt.Sprintf("/api/myshixuns/%s/start.json?%s", url.PathEscape(myshixun), q.Encode())
	var pod podStartResponse
	if err := c.signedRequest(http.MethodGet, path, nil, &pod); err != nil {
		return nil, err
	}
	if pod.Status != 0 {
		return nil, fmt.Errorf("start pod failed: status=%d message=%s", pod.Status, pod.Message)
	}
	return &pod, nil
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func stringValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func numberString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	switch v := m[key].(type) {
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case json.Number:
		return v.String()
	case string:
		return v
	default:
		return ""
	}
}

func arrayValue(m map[string]any, key string) []any {
	v, _ := m[key].([]any)
	return v
}

func statusString(v any) string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return strings.Join(out, ",")
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func courseCode(course map[string]any) string {
	raw := stringValue(course, "first_category_url")
	parts := strings.Split(raw, "/")
	for i, part := range parts {
		if part == "classrooms" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
