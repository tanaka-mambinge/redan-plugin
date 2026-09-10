package main

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/t12e/redan-plugin/internal/auth"
	"github.com/t12e/redan-plugin/internal/client"
)

const loginPagePath = "/redan-login"

// redanLogo is embedded so the login page works without any external files.
//
//go:embed assets/redan-logo.png
var redanLogo []byte

type loginPageData struct {
	Token       string
	Path        string
	Email       string
	Message     string
	Error       bool
	Success     bool
	LogoDataURI template.URL
}

var loginPageTemplate = template.Must(template.New("redan-login").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Sign in to Redan</title>
<style>
:root{color-scheme:light;--blue:#173f9f;--blue-dark:#102d73;--ink:#202124;--muted:#667085;--line:#d0d5dd;--canvas:#f5f7fb;--danger:#a12626;--success:#176b38}
*{box-sizing:border-box}body{margin:0;background:var(--canvas);color:var(--ink);font:16px/1.5 system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}.page{min-height:100dvh;display:grid;place-items:center;padding:24px 16px}.shell{width:min(100%,430px)}.brand{display:grid;place-items:center;padding:20px;background:#fff;border:1px solid #e4e8f0;border-bottom:0;border-radius:18px 18px 0 0}.brand img{display:block;width:100px;height:100px}.panel{padding:30px;background:#fff;border:1px solid rgba(32,33,36,.1);border-top:0;border-radius:0 0 18px 18px;box-shadow:0 18px 42px rgba(32,33,36,.12)}h1{margin:0;font-size:28px;line-height:1.15}.intro{margin:10px 0 24px;color:var(--muted)}.field{margin-top:17px}label{display:block;margin-bottom:7px;font-size:14px;font-weight:650}input{display:block;width:100%;min-height:48px;padding:11px 13px;border:1px solid var(--line);border-radius:10px;background:#fff;color:var(--ink);font:inherit;font-size:16px}input:focus{border-color:var(--blue);outline:2px solid var(--blue);outline-offset:-1px}.status{margin:20px 0 0;padding:12px 14px;border-radius:10px;font-size:14px}.status.error{background:#fff1f0;color:var(--danger)}.status.ok{background:#effaf3;color:var(--success)}button{display:block;width:100%;min-height:50px;margin-top:24px;padding:12px 16px;border:0;border-radius:10px;background:var(--blue);color:#fff;font:inherit;font-weight:650;cursor:pointer}button:hover{background:var(--blue-dark)}.success{text-align:center}.close{margin:12px 0 0;color:var(--muted)}
</style></head><body><main class="page"><section class="shell"><header class="brand"><img src="{{.LogoDataURI}}" width="100" height="100" alt="Redan"></header><div class="panel{{if .Success}} success{{end}}">
{{if .Success}}<h1>You're signed in</h1><p class="intro">Your Redan account is connected.</p><p class="close">You can close this page and return to your request.</p>{{else}}<h1>Connect your Redan account</h1><p class="intro">Sign in with your Redan admin account to manage content.</p>{{if .Message}}<p class="status {{if .Error}}error{{else}}ok{{end}}" role="{{if .Error}}alert{{else}}status{{end}}">{{.Message}}</p>{{end}}<form method="post" action="{{.Path}}?token={{.Token}}" onsubmit="const b=this.querySelector('button');b.disabled=true;b.textContent='Signing in…';"><div class="field"><label for="email">Email address</label><input id="email" name="email" type="email" value="{{.Email}}" autocomplete="username" autocapitalize="none" required autofocus></div><div class="field"><label for="password">Password</label><input id="password" name="password" type="password" autocomplete="current-password" required></div><button type="submit">Sign in securely</button></form>{{end}}</div></section></main></body></html>`))

func runBrowserLogin(ctx context.Context, apiURL, initialEmail string, jsonOutput bool, out io.Writer) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start local login page: %w", err)
	}
	defer listener.Close()
	localToken, err := randomLoginToken()
	if err != nil {
		return fmt.Errorf("create local login token: %w", err)
	}

	resultCh := make(chan error, 1)
	var once sync.Once
	finish := func(result error) { once.Do(func() { resultCh <- result }) }
	var tokenMu sync.RWMutex
	tokenExpires := time.Now().Add(5 * time.Minute)
	validToken := func(candidate string) bool {
		tokenMu.RLock()
		defer tokenMu.RUnlock()
		return localToken != "" && candidate == localToken && time.Now().Before(tokenExpires)
	}
	invalidateToken := func() {
		tokenMu.Lock()
		localToken = ""
		tokenMu.Unlock()
	}

	mux := http.NewServeMux()
	mux.HandleFunc(loginPagePath, func(writer http.ResponseWriter, request *http.Request) {
		if !validToken(request.URL.Query().Get("token")) {
			writeLoginPage(writer, http.StatusNotFound, loginPageData{Message: "Login page not found.", Error: true})
			return
		}
		if request.Method == http.MethodGet {
			writeLoginPage(writer, http.StatusOK, loginPageData{Token: localToken, Email: initialEmail})
			return
		}
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		request.Body = http.MaxBytesReader(writer, request.Body, 64<<10)
		if err := request.ParseForm(); err != nil {
			writeLoginPage(writer, http.StatusBadRequest, loginPageData{Token: localToken, Message: "The submitted form was invalid.", Error: true})
			return
		}
		email := strings.TrimSpace(request.FormValue("email"))
		password := request.FormValue("password")
		if email == "" || password == "" {
			writeLoginPage(writer, http.StatusBadRequest, loginPageData{Token: localToken, Email: email, Message: "Email and password are required.", Error: true})
			return
		}

		result, loginErr := client.Login(request.Context(), apiURL, email, password)
		if loginErr != nil {
			writeLoginPage(writer, http.StatusBadRequest, loginPageData{Token: localToken, Email: email, Message: loginErr.Error(), Error: true})
			return
		}
		if err := auth.NewStore().Save(auth.Credential{
			Token: result.Data.Token, APIURL: strings.TrimRight(apiURL, "/"), ExpiresAt: result.Data.ExpiresAt,
			UserID: result.Data.User.ID, UserName: result.Data.User.Name, UserEmail: result.Data.User.Email,
		}); err != nil {
			writeLoginPage(writer, http.StatusInternalServerError, loginPageData{Token: localToken, Email: email, Message: "The login succeeded, but the session could not be saved in the OS keyring.", Error: true})
			return
		}
		invalidateToken()
		writeLoginPage(writer, http.StatusOK, loginPageData{Success: true})
		finish(nil)
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 30 * time.Second}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			finish(fmt.Errorf("local login page stopped: %w", serveErr))
		}
	}()

	loginURL := "http://" + listener.Addr().String() + loginPagePath + "?" + url.Values{"token": []string{localToken}}.Encode()
	_, _ = fmt.Fprintf(out, "Opening local Redan login page: %s\n", loginURL)
	_ = openLoginBrowser(loginURL)

	loginExpiry := time.NewTimer(5 * time.Minute)
	defer loginExpiry.Stop()
	select {
	case err := <-resultCh:
		shutdownContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownContext)
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSON(out, map[string]any{"authenticated": true, "message": "Redan session saved in the OS keyring"})
		}
		_, err = fmt.Fprintln(out, "Redan login completed; session saved in the OS keyring.")
		return err
	case <-loginExpiry.C:
		_ = server.Shutdown(context.Background())
		return errors.New("local login page expired; run redan auth login again")
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return ctx.Err()
	}
}

func writeLoginPage(writer http.ResponseWriter, status int, data loginPageData) {
	writer.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; img-src data:; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("X-Frame-Options", "DENY")
	writer.Header().Set("Referrer-Policy", "no-referrer")
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	data.Path = loginPagePath
	data.LogoDataURI = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(redanLogo))
	data.Token = template.HTMLEscapeString(data.Token)
	_ = loginPageTemplate.Execute(writer, data)
}

func randomLoginToken() (string, error) {
	data := make([]byte, 24)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func openLoginBrowser(target string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	return command.Start()
}
