package main

import (
	"bytes"
	"testing"
)

func TestCredentialsPrint(t *testing.T) {
	tt := []struct {
		keyId    string
		secretId string
		session  string
	}{
		{"accessKeyId", "secretAccessKey", "session"},
	}

	expected := []struct {
		output string
	}{
		{"export AWS_ACCESS_KEY_ID=accessKeyId\nexport AWS_SECRET_ACCESS_KEY=secretAccessKey\nexport AWS_SESSION_TOKEN=session\n"},
	}

	for i, tc := range tt {
		var output bytes.Buffer
		c := new(credentials)
		c.New(tc.keyId, tc.secretId, tc.session)
		c.Print(&output)

		if output.String() != expected[i].output {
			t.Errorf("got %s but expected %s", output.String(), expected[i].output)
		}
	}

}
