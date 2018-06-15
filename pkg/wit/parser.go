package wit

import (
	"errors"
	"log"
	"time"

	"github.com/fmitra/dennis-bot/pkg/utils"
)

const (
	TRACKING_REQUESTED_SUCCESS = "tracking_requested_success"

	TRACKING_REQUESTED_ERROR = "tracking_requested_error"

	EXPENSE_TOTAL_REQUESTED_SUCCESS = "expense_total_requested_success"

	UNKNOWN_REQUEST = "unknown_request"
)

// Wit.ai Entity
type WitEntity []struct {
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
}

// Wit.ai API Response
type WitResponse struct {
	Entities struct {
		Amount      WitEntity `json:"amount"`
		DateTime    WitEntity `json:"datetime"`
		Description WitEntity `json:"description"`
		TotalSpent  WitEntity `json:"total_spent"`
	} `json:"entities"`
}

// Checks if Wit.ai was able to infer a total spent query
func (w WitResponse) GetSpendPeriod() (string, error) {
	totalSpent := w.Entities.TotalSpent
	if len(totalSpent) == 0 {
		return "", errors.New("No period specified")
	}

	return totalSpent[0].Value, nil
}

// Checks if Wit.ai was able to infer a valid
// Amount Entity from the IncomingMessage.getMessage()
func (w WitResponse) GetAmount() (float64, string, error) {
	amount := w.Entities.Amount
	if len(amount) == 0 {
		return 0, "", errors.New("No amount")
	}

	totalAmount, currency := utils.ParseAmount(amount[0].Value)

	if totalAmount > 0 && currency != "" {
		return totalAmount, currency, nil
	}

	return 0, "", errors.New("Invalid amount")
}

// Checks if Wit.ai was able to infer a valid
// Description Entity from the IncomingMessage.getMessage()
func (w WitResponse) GetDescription() (string, error) {
	description := w.Entities.Description
	if len(description) == 0 {
		return "", errors.New("No description")
	}

	parsedDescription := utils.ParseDescription(description[0].Value)
	return parsedDescription, nil
}

// Checks if Wit.ai was able to infer a valid
// Date Entity from the IncomingMessage.getMessage()
// If no date is provided, it will always default to today
func (w WitResponse) GetDate() time.Time {
	dateTime := w.Entities.DateTime
	stringDate := ""
	if len(dateTime) != 0 {
		stringDate = dateTime[0].Value
	}

	parsedDate := utils.ParseDate(stringDate)
	return parsedDate
}

// Infers whether the User is attempting to track an expense
func (w WitResponse) IsTracking() (bool, error) {
	_, _, err := w.GetAmount()
	if err != nil {
		log.Printf("wit: cannot infer without amount")
		return false, err
	}

	_, err = w.GetDescription()
	if err != nil {
		log.Printf("wit: cannot infer without description")
		return true, err
	}

	return true, nil
}

func (w WitResponse) IsRequestingTotal() (bool, error) {
	_, err := w.GetSpendPeriod()
	if err != nil {
		log.Printf("wit: cannot infer spend period")
		return false, err
	}

	return true, nil
}

// Returns an overview of a message based on Wit.ai's parsing
// We use this to provide context to infer a direction the bot should
// take when talking to a user
func (w WitResponse) GetMessageOverview() string {
	isTracking, trackingErr := w.IsTracking()
	isRequestingTotal, totalErr := w.IsRequestingTotal()

	if isTracking && trackingErr != nil {
		return TRACKING_REQUESTED_ERROR
	} else if isTracking && trackingErr == nil {
		return TRACKING_REQUESTED_SUCCESS
	} else if isRequestingTotal && totalErr == nil {
		return EXPENSE_TOTAL_REQUESTED_SUCCESS
	} else {
		return UNKNOWN_REQUEST
	}
}
