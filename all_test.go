package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestWebhookHandler(t *testing.T) {
	tests := []struct {
		name         string
		payload      WebhookPayload
		properties   map[string]string
		expectedURL  string
		expectedBody string
	}{
		{
			name: "Post to Commit",
			payload: WebhookPayload{
				Branch: Branch{IsMain: false},
				QualityGate: QualityGate{Status: "OK"},
				Properties: map[string]string{
					"sonar.analysis.project_id": "123",
					"sonar.analysis.commit_sha": "abc123",
				},
			},
			properties: map[string]string{
				"GITLAB_URL": "http://gitlab.example.com",
				"GITLAB_TOKEN": "token123",
			},
			expectedURL:  "http://gitlab.example.com/api/v4/projects/123/repository/commits/abc123/comments",
			expectedBody: "note=SonarQube+Quality+Gate%3A+%3Ca+href%3D%27%27%3EOK%3C%2Fa%3E%5Cn%5Cn",
		},
		{
			name: "Post to Merge Request",
			payload: WebhookPayload{
				Branch: Branch{IsMain: false},
				QualityGate: QualityGate{Status: "FAILED"},
				Properties: map[string]string{
					"sonar.analysis.project_id": "123",
					"sonar.analysis.mr_iid": "456",
				},
			},
			properties: map[string]string{
				"GITLAB_URL": "http://gitlab.example.com",
				"GITLAB_TOKEN": "token123",
			},
			expectedURL:  "http://gitlab.example.com/api/v4/projects/123/merge_requests/456/notes",
			expectedBody: "note=SonarQube+Quality+Gate%3A+%3Ca+href%3D%27%27%3EFAILED%3C%2Fa%3E%5Cn%5Cn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock environment variables
			for key, value := range tt.properties {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.properties {
					os.Unsetenv(key)
				}
			}()

			// Mock HTTP client
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.String() != tt.expectedURL {
					t.Errorf("unexpected URL: got %v, want %v", r.URL.String(), tt.expectedURL)
				}
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("failed to read request body: %v", err)
				}
				if string(body) != tt.expectedBody {
					t.Errorf("unexpected body: got %v, want %v", string(body), tt.expectedBody)
				}
				w.WriteHeader(http.StatusCreated)
			}))
			defer server.Close()

			// Override GitLab URL to use mock server
			os.Setenv("GITLAB_URL", server.URL)

			// Create handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload WebhookPayload
				body, _ := json.Marshal(tt.payload)
				r.Body = ioutil.NopCloser(bytes.NewReader(body))
				r.Method = http.MethodPost
				main()
			})

			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			if status := resp.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v, want %v", status, http.StatusOK)
			}
		})
	}
}
