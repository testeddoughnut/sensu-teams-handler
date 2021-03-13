package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageColor(t *testing.T) {
	assert := assert.New(t)
	event := corev2.FixtureEvent("entity1", "check1")

	event.Check.Status = 0
	color := messageColor(event)
	assert.Equal("#008450", color)

	event.Check.Status = 1
	color = messageColor(event)
	assert.Equal("#EFB700", color)

	event.Check.Status = 2
	color = messageColor(event)
	assert.Equal("#B81D13", color)
}

func TestMessageStatus(t *testing.T) {
	assert := assert.New(t)
	event := corev2.FixtureEvent("entity1", "check1")

	event.Check.Status = 0
	status := messageStatus(event)
	assert.Equal("Resolved", status)

	event.Check.Status = 1
	status = messageStatus(event)
	assert.Equal("Warning", status)

	event.Check.Status = 2
	status = messageStatus(event)
	assert.Equal("Critical", status)
}

func TestSendMessage(t *testing.T) {
	assert := assert.New(t)
	event := corev2.FixtureEvent("entity1", "check1")

	var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		expectedBody := "{\"@type\":\"MessageCard\",\"@context\":\"https://schema.org/extensions\",\"summary\":\"Sensu Event: entity1/check1: passing\",\"title\":\"Sensu Event (Resolved)\",\"themeColor\":\"#008450\",\"sections\":[{\"facts\":[{\"name\":\"Entity\",\"value\":\"entity1\"},{\"name\":\"Check\",\"value\":\"check1\"},{\"name\":\"State\",\"value\":\"passing\"},{\"name\":\"Occurrences\",\"value\":\"0\"},{\"name\":\"Output\",\"value\":\"```\\n```\"}]}]}"
		assert.Equal(expectedBody, string(body))
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`1`))
		require.NoError(t, err)
	}))

	config.teamsWebHookURL = apiStub.URL
	config.teamsSummaryTemplate = "Sensu Event: {{.Entity.Name}}/{{.Check.Name}}: {{.Check.State}}"
	err := sendMessage(event)
	assert.NoError(err)
}

func TestMain(t *testing.T) {
	assert := assert.New(t)
	file, _ := ioutil.TempFile(os.TempDir(), "sensu-handler-teams-")
	defer func() {
		_ = os.Remove(file.Name())
	}()

	event := corev2.FixtureEvent("entity1", "check1")
	eventJSON, _ := json.Marshal(event)
	_, err := file.WriteString(string(eventJSON))
	require.NoError(t, err)
	require.NoError(t, file.Sync())
	_, err = file.Seek(0, 0)
	require.NoError(t, err)
	os.Stdin = file
	requestReceived := false

	var apiStub = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`1`))
		require.NoError(t, err)
	}))

	oldArgs := os.Args
	os.Args = []string{"sensu-handler-teams", "-w", apiStub.URL}
	defer func() { os.Args = oldArgs }()

	main()
	assert.True(requestReceived)
}
