package conversation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/fmitra/dennis-bot/internal/actions"
	"github.com/fmitra/dennis-bot/pkg/alphapoint"
	"github.com/fmitra/dennis-bot/pkg/telegram"
	"github.com/fmitra/dennis-bot/pkg/wit"
	mocks "github.com/fmitra/dennis-bot/test"
)

type ConvoSuite struct {
	suite.Suite
	Env *mocks.TestEnv
}

func (suite *ConvoSuite) SetupSuite() {
	configFile := "../../config/config.json"
	suite.Env = mocks.GetTestEnv(configFile)
}

func (suite *ConvoSuite) TearDownSuite() {
	mocks.CleanUpEnv(suite.Env)
}

func (suite *ConvoSuite) BeforeTest(suiteName, testName string) {
	// Responses may be randomized from a list of options.
	// We need to ensure the returned response is predictable
	MessageMap = mocks.MessageMapMock
	mocks.CleanUpEnv(suite.Env)
}

func (suite *ConvoSuite) TestReturnsBooleanForResponseCheck() {
	conversation := &Conversation{}
	hasResponse := conversation.HasResponse()
	assert.True(suite.T(), hasResponse)

	conversation = &Conversation{
		Step: -1,
	}
	hasResponse = conversation.HasResponse()
	assert.False(suite.T(), hasResponse)
}

func (suite *ConvoSuite) TestCreatesNewConversation() {
	rawWitResponse := []byte(`{
		"entities": {
			"amount": [
				{ "value": "20 SGD", "confidence": 100.00 }
			],
			"datetime": [
				{ "value": "", "confidence": 100.00 }
			],
			"description": [
				{ "value": "Food", "confidence": 100.00 }
			]
		}
	}`)
	var witResponse wit.Response
	json.Unmarshal(rawWitResponse, &witResponse)
	action := &actions.Actions{
		Db: suite.Env.Db,
	}

	conversation := NewConversation(mocks.TestUserID, witResponse, action)
	assert.Equal(suite.T(), mocks.TestUserID, conversation.UserID)
	assert.Equal(suite.T(), OnboardUserIntent, conversation.IntentType)
}

func (suite *ConvoSuite) TestInfersUserIntentFromWitResponse() {
	var rawWitResponse []byte
	var witResponse wit.Response

	rawWitResponse = []byte(`{
		"entities": {
			"amount": [
				{ "value": "20 SGD", "confidence": 100.00 }
			],
			"datetime": [
				{ "value": "", "confidence": 100.00 }
			],
			"description": [
				{ "value": "Food", "confidence": 100.00 }
			]
		}
	}`)
	json.Unmarshal(rawWitResponse, &witResponse)
	assert.Equal(suite.T(), OnboardUserIntent, InferIntent(witResponse, uint(0)))

	rawWitResponse = []byte(`{
		"entities": {
			"amount": [
				{ "value": "20 SGD", "confidence": 100.00 }
			],
			"datetime": [
				{ "value": "", "confidence": 100.00 }
			],
			"description": [
				{ "value": "Food", "confidence": 100.00 }
			]
		}
	}`)
	json.Unmarshal(rawWitResponse, &witResponse)
	assert.Equal(suite.T(), TrackExpenseIntent, InferIntent(witResponse, uint(123)))

	rawWitResponse = []byte(`{
		"entities": {
			"amount": [
				{ "value": "20 SGD", "confidence": 100.00 }
			],
			"datetime": [
				{ "value": "", "confidence": 100.00 }
			],
			"description": []
		}
	}`)
	json.Unmarshal(rawWitResponse, &witResponse)
	assert.Equal(suite.T(), TrackExpenseIntent, InferIntent(witResponse, uint(123)))

	rawWitResponse = []byte(`{
		"entities": {
			"amount": [],
			"datetime": [],
			"description": [],
			"total_spent": [
				{ "value": "month", "confidence": 100.00 }
			]
		}
	}`)
	json.Unmarshal(rawWitResponse, &witResponse)
	assert.Equal(suite.T(), GetExpenseTotalIntent, InferIntent(witResponse, uint(123)))

	rawWitResponse = []byte(`{
		"entities": {
			"amount": [],
			"datetime": [],
			"description": [],
			"total_spent": []
		}
	}`)
	json.Unmarshal(rawWitResponse, &witResponse)
	assert.Equal(suite.T(), "", InferIntent(witResponse, uint(123)))
}

func (suite *ConvoSuite) TestGetsConversationFromCache() {
	cache := suite.Env.Cache
	conversation := Conversation{
		IntentType: OnboardUserIntent,
		UserID:     mocks.TestUserID,
	}
	cacheKey := fmt.Sprintf("%s_conversation", strconv.Itoa(int(mocks.TestUserID)))

	oneMinute := 60
	cache.Set(cacheKey, conversation, oneMinute)
	cachedConversation, err := GetConversation(mocks.TestUserID, cache)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), conversation, cachedConversation)
}

func (suite *ConvoSuite) TestReturnsErrorFetchingFromCache() {
	cache := suite.Env.Cache
	cachedConversation, err := GetConversation(mocks.TestUserID, cache)
	assert.EqualError(suite.T(), err, "no conversation found")
	assert.Equal(suite.T(), cachedConversation, Conversation{})

	conversation := Conversation{
		IntentType: OnboardUserIntent,
		UserID:     mocks.TestUserID,
		Step:       -1,
	}
	cacheKey := fmt.Sprintf("%s_conversation", strconv.Itoa(int(mocks.TestUserID)))
	oneMinute := 60
	cache.Set(cacheKey, conversation, oneMinute)
	_, err = GetConversation(mocks.TestUserID, cache)
	assert.EqualError(suite.T(), err, "no responses available")
}

func (suite *ConvoSuite) TestRetrievesIntent() {
	conversation := &Conversation{
		IntentType: OnboardUserIntent,
	}
	intent := conversation.GetIntent(&actions.Actions{})
	assert.IsType(suite.T(), &OnboardUser{}, intent)
}

func (suite *ConvoSuite) TestGetResponseInCorrectOrder() {
	conversation := &Conversation{
		IntentType: OnboardUserIntent,
	}
	a := &actions.Actions{
		Db:         suite.Env.Db,
		Cache:      suite.Env.Cache,
		Config:     suite.Env.Config,
		Alphapoint: &alphapoint.Client{},
	}
	witResponse := wit.Response{}
	incMessage := telegram.IncomingMessage{}
	message := mocks.GetMockMessage("")
	json.Unmarshal(message, &incMessage)

	// Starts at step 0
	assert.Equal(suite.T(), 0, conversation.Step)

	// First response requests password
	conversation.SetLastUserMessage(witResponse, incMessage)
	response := conversation.Respond(a)
	assert.Equal(suite.T(), BotResponse("What's your password?"), response)
	assert.Equal(suite.T(), 1, conversation.Step)

	// Second response requests confirmation
	message = mocks.GetMockMessage("foo")
	json.Unmarshal(message, &incMessage)
	conversation.SetLastUserMessage(witResponse, incMessage)
	response = conversation.Respond(a)
	assert.Equal(suite.T(), BotResponse("Your password is foo"), response)
	assert.Equal(suite.T(), 2, conversation.Step)

	// Invalid response prevents user from reaching step 3
	message = mocks.GetMockMessage("invalid answer")
	json.Unmarshal(message, &incMessage)
	conversation.SetLastUserMessage(witResponse, incMessage)
	response = conversation.Respond(a)
	assert.Equal(suite.T(), BotResponse("I didn't understand that"), response)
	assert.Equal(suite.T(), 2, conversation.Step)

	// Answering no to password confirmation ends the conversation
	message = mocks.GetMockMessage("No")
	json.Unmarshal(message, &incMessage)
	conversation.SetLastUserMessage(witResponse, incMessage)
	response = conversation.Respond(a)
	assert.Equal(suite.T(), BotResponse("Okay try again later"), response)
	assert.Equal(suite.T(), -1, conversation.Step)

	// After receiving a negative step, all future responses are empty
	message = mocks.GetMockMessage("Hello?")
	json.Unmarshal(message, &incMessage)
	conversation.SetLastUserMessage(witResponse, incMessage)
	response = conversation.Respond(a)
	assert.Equal(suite.T(), BotResponse(""), response)
	assert.Equal(suite.T(), -1, conversation.Step)

	// Manually edit the step so we can continue the final tests
	conversation.Step = 5

	// When steps are iterated past the number of responses, we should
	// reset the step to -1 to end the conversation
	message = mocks.GetMockMessage("Yes")
	json.Unmarshal(message, &incMessage)
	conversation.SetLastUserMessage(witResponse, incMessage)
	response = conversation.Respond(a)
	assert.Equal(suite.T(), BotResponse("Outro message"), response)
	assert.Equal(suite.T(), -1, conversation.Step)
}

func (suite *ConvoSuite) TestCachesConversationsWithRemainingResponses() {
	cacheKey := fmt.Sprintf("%s_conversation", strconv.Itoa(int(mocks.TestUserID)))
	a := &actions.Actions{
		Cache: suite.Env.Cache,
		Db:    suite.Env.Db,
	}
	witResponse := wit.Response{}
	incMessage := telegram.IncomingMessage{}
	message := mocks.GetMockMessage("")
	json.Unmarshal(message, &incMessage)

	GetResponse(witResponse, incMessage, a)

	var cachedConvo Conversation
	suite.Env.Cache.Get(cacheKey, &cachedConvo)
	assert.Equal(suite.T(), OnboardUserIntent, cachedConvo.IntentType)
}

func TestConvoSuite(t *testing.T) {
	suite.Run(t, new(ConvoSuite))
}
