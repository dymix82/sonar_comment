package main
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)
type QualityGateCondition struct {
	Metric         string json:"metric"
	Operator       string json:"operator"
	Value          string json:"value"
	Status         string json:"status"
	ErrorThreshold string json:"errorThreshold"
}
type QualityGate struct {
	Name       string                json:"name"
	Status     string                json:"status"
	Conditions []QualityGateCondition json:"conditions"
}
type Project struct {
	Key  string json:"key"
	Name string json:"name"
	URL  string json:"url"
}
type Branch struct {
	Name   string json:"name"
	Type   string json:"type"
	IsMain bool   json:"isMain"
	URL    string json:"url"
}
type WebhookPayload struct {
	ServerURL   string            json:"serverUrl"
	TaskID      string            json:"taskId"
	Status      string            json:"status"
	AnalysedAt  string            json:"analysedAt"
	Revision    string            json:"revision"
	ChangedAt   string            json:"changedAt"
	Project     Project           json:"project"
	Branch      Branch            json:"branch"
	QualityGate QualityGate       json:"qualityGate"
	Properties  map[string]string  json:"properties"
}
func main() {
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}
		var payload WebhookPayload
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		err = json.Unmarshal(body, &payload)
		if err != nil {
			http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
			return
		}
		// Логика для основной ветки
		if payload.Branch.IsMain {
			log.Println("Skipping main branch")
			return
		}
		// Данные для комментария
		comment := formatComment(payload.QualityGate, payload.Branch)
		// URL и токен GitLab
		gitlabURL := os.Getenv("GITLAB_URL")
		token := os.Getenv("GITLAB_TOKEN")
		projectID := payload.Properties["sonar.analysis.project_id"]
		commitSHA := payload.Properties["sonar.analysis.commit_sha"]
		if projectID == "" || commitSHA == "" {
			log.Println("Missing required properties for MR")
			return
		}
		commentURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits/%s/comments", gitlabURL, projectID, commitSHA)
		// Отправка комментария в GitLab
		err = postToGitLab(commentURL, comment, token)
		if err != nil {
			log.Printf("Failed to post comment: %v\n", err)
		} else {
			log.Println("Comment posted successfully")
		}
	})
	port := ":8080"
	log.Printf("Listening on %s...\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
func formatComment(qualityGate QualityGate, branch Branch) string {
	comment := fmt.Sprintf("SonarQube Quality Gate: <a href='%s'>%s</a>\n\n", branch.URL, qualityGate.Status)
	for _, condition := range qualityGate.Conditions {
		comment += fmt.Sprintf(
			"- %s (%s): %s (%s) [Threshold: %s]\n",
			condition.Metric,
			condition.Operator,
			condition.Value,
			condition.Status,
			condition.ErrorThreshold,
		)
	}
	return comment
}
func postToGitLab(urlStr, comment, token string) error {
	// Создаем тело запроса с использованием url.Values
	data := url.Values{}
	data.Set("note", comment)
	// Создаем HTTP-запрос
	req, err := http.NewRequest("POST", urlStr, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}
	// Добавляем заголовки
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Выполняем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Проверяем статус ответа
	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to post comment: %s", string(body))
	}
	return nil
}