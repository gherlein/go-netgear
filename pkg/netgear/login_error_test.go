package netgear

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// gs316LoginPage mirrors the login page a GS316EPP returns, with errorMessage
// in the span its firmware uses to report why a login was refused.
func gs316LoginPage(errorMessage string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><head><title>NETGEAR  GS316EPP</title></head>
<body id="loginBody"><form name="login" method="post" action="/redirect.html">
<input type="hidden" id="submitPwd" name="LoginPassword" value="">
<input id="Password" type="password" value="" maxlength="20">
<input type="hidden" id='rand' value="2085997689" disabled>
<span id="loginPageErrorMsg" class="validationRed">%s</span>
</form></body></html>`, errorMessage)
}

// newGS316LoginServer serves the login page on every GET and answers the
// login POST with a login page carrying refusal.
func newGS316LoginServer(t *testing.T, refusal string) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			fmt.Fprint(w, gs316LoginPage(refusal))
			return
		}
		fmt.Fprint(w, gs316LoginPage(""))
	}))
	t.Cleanup(server.Close)
	address := strings.TrimPrefix(server.URL, "http://")
	client, err := NewClient(address,
		WithTokenManager(NewMemoryTokenManager()),
		WithEnvironmentAuth(false))
	if err != nil {
		t.Fatal(err)
	}
	if client.GetModel() != ModelGS316EPP {
		t.Fatalf("detected model %q, want %q", client.GetModel(), ModelGS316EPP)
	}
	return client
}

func TestGambitLoginReportsSwitchRefusal(t *testing.T) {
	const refusal = "The maximum number of sessions has been reached. Wait a few minutes and then try again."
	client := newGS316LoginServer(t, refusal)
	err := client.Login(context.Background(), "password")
	if err == nil {
		t.Fatal("login succeeded, want refusal")
	}
	if !strings.Contains(err.Error(), refusal) {
		t.Fatalf("error %q does not carry the switch's message", err)
	}
	if errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error %q reported as invalid credentials", err)
	}
	var netgearError *Error
	if !errors.As(err, &netgearError) || netgearError.Type != ErrorTypeAuth {
		t.Fatalf("error %q is not an auth error", err)
	}
}

func TestGambitLoginWithoutMessageIsInvalidCredentials(t *testing.T) {
	client := newGS316LoginServer(t, "")
	err := client.Login(context.Background(), "password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}
