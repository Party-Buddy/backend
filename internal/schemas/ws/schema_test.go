package ws

import (
	"encoding/json"
	"testing"
)

func Test_MessageTaskAnswer_WithChoiceAnswer_Deserialized(t *testing.T) {
	jsonStr := `
		{
			"msg-id": 1,
			"kind": "task-answer",
			"time": 1701515325249,
			"ready": true,
			"task-idx": 0,
			"answer": {
				"type": "option",
				"value": 3
			}
		}
	`
	var a MessageTaskAnswer
	err := json.Unmarshal([]byte(jsonStr), &a)
	if err != nil {
		t.Fatalf("fail to deserialize MessageTaskAnswer with err: %v", err)
	}
}

func Test_MessageTaskAnswer_WithTextAnswer_Deserialized(t *testing.T) {
	jsonStr := `
		{
			"msg-id": 1,
			"kind": "task-answer",
			"time": 1701515325249,
			"ready": true,
			"task-idx": 0,
			"answer": {
				"type": "text",
				"value": "hello world"
			}
		}
	`
	var a MessageTaskAnswer
	err := json.Unmarshal([]byte(jsonStr), &a)
	if err != nil {
		t.Fatalf("fail to deserialize MessageTaskAnswer with err: %v", err)
	}
}

func Test_MessageTaskAnswer_WithCheckedTextAnswer_Deserialized(t *testing.T) {
	jsonStr := `
		{
			"msg-id": 1,
			"kind": "task-answer",
			"time": 1701515325249,
			"ready": true,
			"task-idx": 0,
			"answer": {
				"type": "checked-text",
				"value": "hello world"
			}
		}
	`
	var a MessageTaskAnswer
	err := json.Unmarshal([]byte(jsonStr), &a)
	if err != nil {
		t.Fatalf("fail to deserialize MessageTaskAnswer with err: %v", err)
	}
}
