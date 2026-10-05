package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type profileRoundTripFunc func(*http.Request) (*http.Response, error)

func (f profileRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAdmissionProfileOfficialResponses(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		missing    bool
		valid      bool
	}{
		{"exists", `{"id":"069a79f444e94726a5befca90e38aaf5","name":"Notch"}`, 200, false, true},
		{"not-found", `{}`, 404, true, false},
		{"no-content", ``, 204, true, false},
		{"rate-limit", `{}`, 429, false, false},
		{"server-failure", `{}`, 503, false, false},
		{"invalid-json", `broken`, 200, false, false},
		{"wrong-name", `{"id":"069a79f444e94726a5befca90e38aaf5","name":"SomeoneElse"}`, 200, false, false},
		{"invalid-uuid", `{"id":"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx","name":"Notch"}`, 200, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := admissionProfileClient
			t.Cleanup(func() { admissionProfileClient = before })
			admissionProfileClient = &http.Client{Transport: profileRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "https://api.minecraftservices.com/minecraft/profile/lookup/name/notch" {
					t.Fatal("lookup left official endpoint", r.URL)
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})}
			profile, err := (&Server{}).resolveAdmissionProfile(context.Background(), "notch")
			if tc.valid {
				if err != nil || profile.Name != "Notch" || profile.UUID != "069a79f4-44e9-4726-a5be-fca90e38aaf5" {
					t.Fatalf("valid profile not resolved: %v %v", profile, err)
				}
			} else if err == nil || errors.Is(err, errAdmissionPlayerMissing) != tc.missing {
				t.Fatalf("incorrect failure classification: %v", err)
			}
		})
	}
}

func TestAdmissionProfileCanceledRequest(t *testing.T) {
	before := admissionProfileClient
	t.Cleanup(func() { admissionProfileClient = before })
	admissionProfileClient = &http.Client{Transport: profileRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (&Server{}).resolveAdmissionProfile(ctx, "Notch")
	if !errors.Is(err, context.Canceled) || errors.Is(err, errAdmissionPlayerMissing) {
		t.Fatalf("cancellation misclassified: %v", err)
	}
}
