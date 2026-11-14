package netgear

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
)

const gs316TestToken = "qwertyuiopasdfghjklzxcvbnmqwertyuiopasdf"

// gs316Switch is a fake GS316EPP serving pages captured from a real switch
// (firmware pages with the session token removed) and recording every
// request so tests can assert the exact form the firmware's own UI sends.
type gs316Switch struct {
	t            *testing.T
	statusPage   string
	settingsPage string
	postReply    string

	mu       sync.Mutex
	requests []recordedRequest
}

type recordedRequest struct {
	method  string
	path    string
	form    url.Values
	rawBody string
}

func newGS316Switch(t *testing.T) *gs316Switch {
	t.Helper()
	return &gs316Switch{
		t:            t,
		statusPage:   readFixture(t, "testdata/gs316epp/poePortStatus.html"),
		settingsPage: readFixture(t, "testdata/gs316epp/poePortConf.html"),
		postReply:    "SUCCESS",
	}
}

func readFixture(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func (s *gs316Switch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Keep the raw body: this firmware reads form fields by position, so
	// the order on the wire matters, not just the decoded values.
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		s.t.Errorf("read body: %v", err)
		return
	}
	form := r.URL.Query()
	if r.Method == http.MethodPost {
		form, err = url.ParseQuery(string(rawBody))
		if err != nil {
			s.t.Errorf("parse form: %v", err)
			return
		}
	}
	s.mu.Lock()
	s.requests = append(s.requests, recordedRequest{method: r.Method, path: r.URL.Path, form: form, rawBody: string(rawBody)})
	s.mu.Unlock()

	if form.Get("Gambit") != gs316TestToken {
		fmt.Fprint(w, gs316LoginPage(""))
		return
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/iss/specific/poePortStatus.html":
		// Without GetData=TRUE this firmware returns an unfilled template.
		if form.Get("GetData") != "TRUE" {
			http.Error(w, "missing GetData", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, s.statusPage)
	case r.Method == http.MethodGet && r.URL.Path == "/iss/specific/poePortConf.html":
		fmt.Fprint(w, s.settingsPage)
	case r.Method == http.MethodPost && r.URL.Path == "/iss/specific/poePortConf.html":
		fmt.Fprint(w, s.postReply)
	case r.Method == http.MethodGet && r.URL.Path == "/iss/specific/logout.html":
		fmt.Fprint(w, `<html><script>top.location.href = "/wmi/login";</script></html>`)
	default:
		http.NotFound(w, r)
	}
}

func (s *gs316Switch) posts() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	var posts []recordedRequest
	for _, request := range s.requests {
		if request.method == http.MethodPost {
			posts = append(posts, request)
		}
	}
	return posts
}

// newGS316Client returns a client already holding a session token, as after
// a cached login, so no model detection or login traffic is involved.
func newGS316Client(t *testing.T, fake *gs316Switch) (*Client, *MemoryTokenManager, string) {
	t.Helper()
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	address := strings.TrimPrefix(server.URL, "http://")
	tokens := NewMemoryTokenManager()
	if err := tokens.StoreToken(context.Background(), address, gs316TestToken, ModelGS316EPP); err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(address, WithTokenManager(tokens), WithEnvironmentAuth(false))
	if err != nil {
		t.Fatal(err)
	}
	return client, tokens, address
}

func TestGS316StatusParsesRealPage(t *testing.T) {
	client, _, _ := newGS316Client(t, newGS316Switch(t))
	statuses, err := client.POE().GetStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 15 {
		t.Fatalf("got %d ports, want 15", len(statuses))
	}
	for index, status := range statuses {
		if status.PortID != index+1 {
			t.Fatalf("entry %d has port %d", index, status.PortID)
		}
	}
	want := POEPortStatus{
		PortID:       1,
		Status:       "Searching",
		PowerClass:   "Unknown",
		VoltageV:     0,
		CurrentMA:    0,
		PowerW:       0,
		TemperatureC: 48,
		ErrorStatus:  "No Error",
	}
	if statuses[0] != want {
		t.Fatalf("port 1 = %+v, want %+v", statuses[0], want)
	}
}

func TestGS316GetPortStatusFindsPort(t *testing.T) {
	client, _, _ := newGS316Client(t, newGS316Switch(t))
	status, err := client.POE().GetPortStatus(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if status.PortID != 2 {
		t.Fatalf("got port %d", status.PortID)
	}
}

func TestGS316StatusRejectsPageWithoutPorts(t *testing.T) {
	fake := newGS316Switch(t)
	fake.statusPage = "<html><body>no ports here</body></html>"
	client, _, _ := newGS316Client(t, fake)
	_, err := client.POE().GetStatus(context.Background())
	var netgearError *Error
	if !errors.As(err, &netgearError) || netgearError.Type != ErrorTypeParsing {
		t.Fatalf("got %v, want a parsing error", err)
	}
}

func TestGS316StatusRejectsUnparseableNumber(t *testing.T) {
	fake := newGS316Switch(t)
	fake.statusPage = strings.Replace(fake.statusPage,
		`<p class="bold-title Temperature-text">48</p>`,
		`<p class="bold-title Temperature-text">TEMPERATURE_KEY</p>`, 1)
	client, _, _ := newGS316Client(t, fake)
	_, err := client.POE().GetStatus(context.Background())
	if err == nil || !strings.Contains(err.Error(), "TEMPERATURE_KEY") {
		t.Fatalf("got %v, want an error naming the bad value", err)
	}
}

func TestGS316SettingsParsesRealPage(t *testing.T) {
	client, _, _ := newGS316Client(t, newGS316Switch(t))
	settings, err := client.POE().GetSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(settings) != 15 {
		t.Fatalf("got %d ports, want 15", len(settings))
	}
	for index, setting := range settings {
		if setting.PortID != index+1 {
			t.Fatalf("entry %d has port %d", index, setting.PortID)
		}
	}
	want := POEPortSettings{
		PortID:              1,
		Enabled:             true,
		Mode:                POEMode8023at,
		Priority:            POEPriorityLow,
		PowerLimitType:      POELimitTypeUser,
		PowerLimitW:         30.0,
		DetectionType:       "IEEE802",
		LongerDetectionTime: false,
	}
	if settings[0] != want {
		t.Fatalf("port 1 = %+v, want %+v", settings[0], want)
	}
}

func TestGS316SettingsReadsDisabledPort(t *testing.T) {
	fake := newGS316Switch(t)
	marker := `<span class="bold-title port-number">2</span>`
	index := strings.Index(fake.settingsPage, marker)
	if index < 0 {
		t.Fatal("fixture has no port 2")
	}
	rest := strings.Replace(fake.settingsPage[index:],
		`<span  width="30%" class="bold-title admin-state">Enable</span>`,
		`<span  width="30%" class="bold-title admin-state">Disable</span>`, 1)
	fake.settingsPage = fake.settingsPage[:index] + rest
	client, _, _ := newGS316Client(t, fake)
	setting, err := client.POE().GetPortSettings(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if setting.Enabled {
		t.Fatal("port 2 reported enabled")
	}
}

func TestGS316DisablePortSendsSubmitPoe(t *testing.T) {
	fake := newGS316Switch(t)
	client, _, _ := newGS316Client(t, fake)
	if err := client.POE().DisablePort(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	posts := fake.posts()
	if len(posts) != 1 {
		t.Fatalf("got %d POSTs, want 1", len(posts))
	}
	// Same fields in the same order as the switch's own UI (submitPoEports).
	want := "Gambit=" + gs316TestToken + "&TYPE=submitPoe&PORT_NO=2&POWER_LIMIT_VALUE=NOTSET" +
		"&PRIORITY=NOTSET&POWER_MODE=NOTSET&POWER_LIMIT_TYPE=NOTSET&DETECTION=NOTSET" +
		"&ADMIN_STATE=0&DISCONNECT_TYPE=NOTSET"
	if posts[0].rawBody != want {
		t.Fatalf("body\n got %s\nwant %s", posts[0].rawBody, want)
	}
	if posts[0].path != "/iss/specific/poePortConf.html" {
		t.Fatalf("posted to %s", posts[0].path)
	}
}

func TestGS316EnablePortSendsAdminStateOne(t *testing.T) {
	fake := newGS316Switch(t)
	client, _, _ := newGS316Client(t, fake)
	if err := client.POE().EnablePort(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if got := fake.posts()[0].form.Get("ADMIN_STATE"); got != "1" {
		t.Fatalf("ADMIN_STATE = %q, want 1", got)
	}
}

func TestGS316UpdateSendsOneRequestPerPort(t *testing.T) {
	fake := newGS316Switch(t)
	client, _, _ := newGS316Client(t, fake)
	off := false
	err := client.POE().UpdatePort(context.Background(),
		POEPortUpdate{PortID: 2, Enabled: &off},
		POEPortUpdate{PortID: 5, Enabled: &off})
	if err != nil {
		t.Fatal(err)
	}
	posts := fake.posts()
	if len(posts) != 2 || posts[0].form.Get("PORT_NO") != "2" || posts[1].form.Get("PORT_NO") != "5" {
		t.Fatalf("posts = %+v", posts)
	}
}

func TestGS316UpdateReportsSwitchRefusal(t *testing.T) {
	fake := newGS316Switch(t)
	fake.postReply = "ONSCHE"
	client, _, _ := newGS316Client(t, fake)
	err := client.POE().DisablePort(context.Background(), 2)
	if err == nil || !strings.Contains(err.Error(), "schedule") {
		t.Fatalf("got %v, want a PoE schedule error", err)
	}
}

func TestGS316UpdateRejectsFailureReply(t *testing.T) {
	fake := newGS316Switch(t)
	fake.postReply = "FAILURE"
	client, _, _ := newGS316Client(t, fake)
	err := client.POE().DisablePort(context.Background(), 2)
	if err == nil || !strings.Contains(err.Error(), "FAILURE") {
		t.Fatalf("got %v, want the switch's reply in the error", err)
	}
}

func TestGS316UpdateRejectsUnsupportedFields(t *testing.T) {
	fake := newGS316Switch(t)
	client, _, _ := newGS316Client(t, fake)
	priority := POEPriorityHigh
	err := client.POE().UpdatePort(context.Background(), POEPortUpdate{PortID: 2, Priority: &priority})
	if err == nil {
		t.Fatal("update with an unsupported field succeeded")
	}
	if len(fake.posts()) != 0 {
		t.Fatal("unsupported update reached the switch")
	}
}

func TestGS316UpdateRejectsPortOutOfRange(t *testing.T) {
	for _, port := range []int{0, 16} {
		fake := newGS316Switch(t)
		client, _, _ := newGS316Client(t, fake)
		if err := client.POE().DisablePort(context.Background(), port); err == nil {
			t.Fatalf("port %d accepted", port)
		}
		if len(fake.posts()) != 0 {
			t.Fatalf("port %d reached the switch", port)
		}
	}
}

func TestGS316CyclePowerSendsResetPoe(t *testing.T) {
	fake := newGS316Switch(t)
	client, _, _ := newGS316Client(t, fake)
	if err := client.POE().CyclePower(context.Background(), 2, 15); err != nil {
		t.Fatal(err)
	}
	posts := fake.posts()
	if len(posts) != 1 {
		t.Fatalf("got %d POSTs, want 1", len(posts))
	}
	want := "Gambit=" + gs316TestToken + "&TYPE=resetPoe&PoePort=010000000000001"
	if posts[0].rawBody != want {
		t.Fatalf("body\n got %s\nwant %s", posts[0].rawBody, want)
	}
}

func TestGS316CyclePowerRejectsPortOutOfRange(t *testing.T) {
	fake := newGS316Switch(t)
	client, _, _ := newGS316Client(t, fake)
	if err := client.POE().CyclePower(context.Background(), 16); err == nil {
		t.Fatal("port 16 accepted")
	}
	if len(fake.posts()) != 0 {
		t.Fatal("rejected cycle reached the switch")
	}
}

func TestGS316LogoutEndsSessionOnSwitch(t *testing.T) {
	fake := newGS316Switch(t)
	client, tokens, address := newGS316Client(t, fake)
	if err := client.Logout(context.Background()); err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	requests := append([]recordedRequest(nil), fake.requests...)
	fake.mu.Unlock()
	if len(requests) != 1 || requests[0].path != "/iss/specific/logout.html" || requests[0].form.Get("Gambit") != gs316TestToken {
		t.Fatalf("requests = %+v, want one logout carrying the token", requests)
	}
	if client.IsAuthenticated() {
		t.Fatal("client still authenticated")
	}
	if _, _, err := tokens.GetToken(context.Background(), address); err == nil {
		t.Fatal("token still cached")
	}
}
