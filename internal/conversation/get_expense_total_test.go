package conversation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/fmitra/dennis-bot/pkg/telegram"
	"github.com/fmitra/dennis-bot/pkg/wit"
	mocks "github.com/fmitra/dennis-bot/test"
)

type ExpenseTotalSuite struct {
	suite.Suite
	Env    *mocks.TestEnv
	Action *Actions
}

func (suite *ExpenseTotalSuite) SetupSuite() {
	configFile := "../../config/config.json"
	suite.Env = mocks.GetTestEnv(configFile)
	suite.Action = &Actions{
		suite.Env.Db,
		suite.Env.Cache,
		suite.Env.Config,
	}
}

func (suite *ExpenseTotalSuite) BeforeTest(suiteName, testName string) {
	MessageMap = mocks.MessageMapMock
}

func (suite *ExpenseTotalSuite) AfterTest(suiteName, testName string) {
	mocks.CleanUpEnv(suite.Env)
}

func (suite *ExpenseTotalSuite) TestGetResponseList() {
	expenseTotal := &GetExpenseTotal{}
	assert.Equal(suite.T(), 3, len(expenseTotal.GetResponses()))
}

func (suite *ExpenseTotalSuite) TestGetExpenseTotalMessage() {
	rawWitResponse := []byte(`{
		"entities": {
			"amount": [],
			"datetime": [],
			"description": [],
			"total_spent": [
				{ "value": "month", "confidence": 100.00 }
			]
		}
	}`)
	var witResponse wit.Response
	json.Unmarshal(rawWitResponse, &witResponse)

	var incMessage telegram.IncomingMessage
	message := mocks.GetMockMessage("")
	json.Unmarshal(message, &incMessage)

	expenseTotal := &GetExpenseTotal{
		Context{
			Step:        2,
			WitResponse: witResponse,
			IncMessage:  incMessage,
			AuxData:     "month",
		},
		suite.Action,
	}
	response, cx := expenseTotal.Respond()
	assert.Equal(suite.T(), BotResponse("You spent 0.00"), response)
	assert.Equal(suite.T(), -1, cx.Step)
}

func (suite *ExpenseTotalSuite) TestGetExpenseTotalError() {
	rawWitResponse := []byte(`{
		"entities": {
			"amount": [],
			"datetime": [],
			"description": [],
			"total_spent": [
				{ "value": "foo", "confidence": 100.00 }
			]
		}
	}`)
	var witResponse wit.Response
	json.Unmarshal(rawWitResponse, &witResponse)

	var incMessage telegram.IncomingMessage
	message := mocks.GetMockMessage("")
	json.Unmarshal(message, &incMessage)

	expenseTotal := &GetExpenseTotal{
		Context{
			Step:        2,
			WitResponse: witResponse,
			IncMessage:  incMessage,
		},
		suite.Action,
	}
	response, cx := expenseTotal.Respond()
	assert.Equal(suite.T(), BotResponse("Whoops!"), response)
	assert.Equal(suite.T(), -1, cx.Step)
}

func (suite *ExpenseTotalSuite) TestAskForPassword() {
	var incMessage telegram.IncomingMessage
	message := mocks.GetMockMessage("")
	json.Unmarshal(message, &incMessage)

	rawWitResponse := []byte(`{
		"entities": {
			"amount": [],
			"datetime": [],
			"description": [],
			"total_spent": [
				{ "value": "month", "confidence": 100.00 }
			]
		}
	}`)
	var witResponse wit.Response
	json.Unmarshal(rawWitResponse, &witResponse)

	expenseTotal := &GetExpenseTotal{
		Context{
			Step:        0,
			IncMessage:  incMessage,
			WitResponse: witResponse,
		},
		suite.Action,
	}

	response, err := expenseTotal.AskForPassword()
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), BotResponse("I need your password"), response)
}

func (suite *ExpenseTotalSuite) TestSkipsPasswordRequest() {
	var incMessage telegram.IncomingMessage
	message := mocks.GetMockMessage("")
	json.Unmarshal(message, &incMessage)

	rawWitResponse := []byte(`{
		"entities": {
			"amount": [],
			"datetime": [],
			"description": [],
			"total_spent": [
				{ "value": "month", "confidence": 100.00 }
			]
		}
	}`)
	var witResponse wit.Response
	json.Unmarshal(rawWitResponse, &witResponse)

	cacheKey := fmt.Sprintf("%s_password", strconv.Itoa(int(incMessage.GetUser().ID)))
	suite.Action.Cache.Set(cacheKey, "my-password", 180)

	expenseTotal := &GetExpenseTotal{
		Context{
			Step:        0,
			IncMessage:  incMessage,
			WitResponse: witResponse,
		},
		suite.Action,
	}

	response, err := expenseTotal.AskForPassword()
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), BotResponse(""), response)
}

func (suite *ExpenseTotalSuite) TestValidatesPassword() {
	var incMessage telegram.IncomingMessage
	message := mocks.GetMockMessage("my-password")
	json.Unmarshal(message, &incMessage)

	rawWitResponse := []byte(`{
		"entities": {
			"amount": [],
			"datetime": [],
			"description": [],
			"total_spent": [
				{ "value": "month", "confidence": 100.00 }
			]
		}
	}`)
	var witResponse wit.Response
	json.Unmarshal(rawWitResponse, &witResponse)

	mocks.CreateTestUser(suite.Env.Db)
	expenseTotal := &GetExpenseTotal{
		Context{
			Step:        0,
			IncMessage:  incMessage,
			WitResponse: witResponse,
		},
		suite.Action,
	}

	response, err := expenseTotal.ValidatePassword()
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), BotResponse(""), response)
}

func (suite *ExpenseTotalSuite) TestSkipsPasswordValidation() {
	var incMessage telegram.IncomingMessage
	message := mocks.GetMockMessage("")
	json.Unmarshal(message, &incMessage)

	cacheKey := fmt.Sprintf("%s_password", strconv.Itoa(int(incMessage.GetUser().ID)))
	suite.Action.Cache.Set(cacheKey, "my-password", 180)

	expenseTotal := &GetExpenseTotal{
		Context{
			Step:       0,
			IncMessage: incMessage,
		},
		suite.Action,
	}

	response, err := expenseTotal.ValidatePassword()
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), BotResponse(""), response)
}

func (suite *ExpenseTotalSuite) TestFailsPasswordValidation() {
	var incMessage telegram.IncomingMessage
	message := mocks.GetMockMessage("Invalid password")
	json.Unmarshal(message, &incMessage)

	mocks.CreateTestUser(suite.Env.Db)
	expenseTotal := &GetExpenseTotal{
		Context{
			Step:       0,
			IncMessage: incMessage,
		},
		suite.Action,
	}

	response, err := expenseTotal.ValidatePassword()
	assert.EqualError(suite.T(), err, "password invalid")
	assert.Equal(suite.T(), BotResponse("This password is invalid"), response)
}

func TestExpenseTotalSuite(t *testing.T) {
	suite.Run(t, new(ExpenseTotalSuite))
}
