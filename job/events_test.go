package job

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/packethost/pkg/log"
	"github.com/tinkerbell/boots/client"
	"github.com/tinkerbell/boots/client/cacher"
	"github.com/tinkerbell/boots/client/packet"
)

func TestPhoneHome(t *testing.T) {
	var reqs []req
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()

		switch r.Method {
		case http.MethodPost, http.MethodPatch:
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
		reqs = append(reqs, req{r.Method, r.URL.String(), string(body)})
		fmt.Println()

		w.Write([]byte(`{"id":"event-id"}`))
	}))
	defer ts.Close()
	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	for name, test := range phoneHomeTests {
		t.Run(name, func(t *testing.T) {
			l := log.Test(t, "PhoneHomeTest")
			reporter, _ := packet.NewReporter(l, u, "", "")
			if err != nil {
				t.Fatal(err)
			}

			reqs = nil

			instance := &client.Instance{
				ID: test.id,
				OSV: &client.OperatingSystem{
					OsSlug: test.os,
				},
			}
			j := Job{
				Logger: joblog.With("test", name),
				mode:   modeInstance,
				hardware: &cacher.HardwareCacher{
					ID:       "$hardware_id",
					State:    client.HardwareState(test.state),
					Instance: instance,
				},
				instance: instance,
				reporter: reporter,
			}
			bad := !j.phoneHome(context.Background(), []byte(test.event))
			if bad != test.bad {
				t.Fatalf("mismatch in expected return from phoneHome, want:%t, got:%t", test.bad, bad)
			}
			if bad {
				return
			}

			if len(test.reqs) != len(reqs) {
				t.Fatalf("mismatch of api requests want:%d got:%d", len(test.reqs), len(reqs))
			}
			for i := range reqs {
				want := test.reqs[i]
				got := reqs[i]
				if want.url != got.url {
					t.Fatalf("mismatch of url in api request want:%q, got:%q", want.url, got.url)
				}
				if want.body != got.body {
					t.Fatalf("mismatch of body in api request want:%q, got:%q", want.body, got.body)
				}
			}
		})
	}
}

type (
	req  struct{ method, url, body string }
	reqs []req
)

var phoneHomeTests = map[string]struct {
	id    string
	event string
	reqs  reqs
	os    string
	bad   bool
	state string
}{
	"bad body": {
		id:    "$instance_id",
		event: "{",
		bad:   true,
	},
	"empty body": {
		id:    "$instance_id",
		event: "",
		reqs:  reqs{{"POST", "/devices/$instance_id/phone-home", ""}},
	},
	"custom_ipxe done": {
		id:    "$instance_id",
		event: `{"type":"provisioning.104.01"}`,
		os:    "custom_ipxe",
		reqs: reqs{
			{"POST", "/devices/$instance_id/events", `{"type":"provisioning.104.01"}`},
			{"PATCH", "/devices/$instance_id", `{"allow_pxe":false}`},
			{"POST", "/devices/$instance_id/phone-home", ``},
		},
	},
	"no id, not preinstalling": {
		event: `{"type":"provisioning.104.01"}`,
		bad:   true,
	},
	"preinstalling": {
		state: "preinstalling",
		event: `{"type":"provisioning.109"}`,
		reqs: reqs{
			{"POST", "/hardware/$hardware_id/events", `{"type":"provisioning.109"}`},
		},
	},
}
