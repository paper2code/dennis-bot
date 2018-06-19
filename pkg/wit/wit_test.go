package wit

import (
	"testing"

	mocks "github.com/fmitra/dennis-bot/test"
	"github.com/stretchr/testify/assert"
)

func TestWit(t *testing.T) {
	t.Run("Returns client with default config", func(t *testing.T) {
		witAi := NewClient("witAiToken")

		assert.Equal(t, BaseUrl, witAi.BaseUrl)
		assert.Equal(t, ApiVersion, witAi.ApiVersion)
	})

	t.Run("Returns WitResponse", func(t *testing.T) {
		response := `{
			"entities": {
				"amount": [
					{ "value": "20 USD", "confidence": 100.00 }
				],
				"datetime": [
					{ "value": "", "confidence": 100.00 }
				],
				"description": [
					{ "value": "Food", "confidence": 100.00 }
				]
			}
		}`
		server := mocks.MakeTestServer(response)
		defer server.Close()

		witAi := Client{
			Token:      "witAiToken",
			BaseUrl:    server.URL,
			ApiVersion: "20180128",
		}

		witResponse := witAi.ParseMessage("Hello world")
		assert.IsType(t, WitResponse{}, witResponse)
	})

	t.Run("Returns zero value WitResponse on error", func(t *testing.T) {
		server := mocks.MakeTestServer(`{not valid json}`)
		defer server.Close()

		witAi := Client{
			Token:      "witAiToken",
			BaseUrl:    server.URL,
			ApiVersion: "20180128",
		}

		witResponse := witAi.ParseMessage("Hello world")
		assert.Equal(t, WitResponse{}, witResponse)
	})
}