package models

import "time"

// Supported bill frequencies.
const (
	BillFrequencyWeekly  = "weekly"
	BillFrequencyMonthly = "monthly"
	BillFrequencyYearly  = "yearly"
)

// Supported bill schedule rule types and scopes.
const (
	BillScheduleTypeWeekday     = "weekday"
	BillScheduleTypeDayOfPeriod = "day_of_period"
	BillScheduleTypeNthWeekday  = "nth_weekday"
	BillScheduleTypeMonthDay    = "month_day"

	BillScheduleScopeWeek  = "week"
	BillScheduleScopeMonth = "month"
	BillScheduleScopeYear  = "year"
)

// Computed bill due statuses.
const (
	BillStatusOverdue     = "overdue"
	BillStatusUpcoming    = "upcoming"
	BillStatusScheduled   = "scheduled"
	BillStatusUnscheduled = "unscheduled"
)

// BillSchedule defines a structured recurring rule constrained by its owning Bill frequency.
// When nil on a Bill, due dates advance purely by Frequency from LastDatePaid (with
// end-of-month and leap-year clamping). When set, Scope must match Frequency
// ("week", "month", or "year"), and calendar-day rules clamp short months/non-leap years.
type BillSchedule struct {
	Type    string `json:"type"`
	Scope   string `json:"scope"`
	Nth     *int   `json:"nth,omitempty"`
	Weekday string `json:"weekday,omitempty"`
	Day     *int   `json:"day,omitempty"`
	Month   *int   `json:"month,omitempty"`
}

// Bill represents a recurring bill owned by a single user.
type Bill struct {
	ID           string        `json:"id"`
	UserID       string        `json:"user_id"`
	AccountID    *string       `json:"account_id"`
	CategoryID   *string       `json:"category_id"`
	Amount       float64       `json:"amount"`
	LastDatePaid *string       `json:"last_date_paid"`
	Frequency    string        `json:"frequency"`
	Interval     int           `json:"interval"`
	Schedule     *BillSchedule `json:"schedule"`
	Name         string        `json:"name"`
	Recipient    string        `json:"recipient"`
	Description  string        `json:"description"`
	Reminder     bool          `json:"reminder"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// BillCreateRequest represents the payload to create a recurring bill.
type BillCreateRequest struct {
	AccountID    *string       `json:"account_id,omitempty"`
	CategoryID   *string       `json:"category_id,omitempty"`
	Amount       *float64      `json:"amount" binding:"required"`
	LastDatePaid *string       `json:"last_date_paid,omitempty"`
	Frequency    string        `json:"frequency" binding:"required"`
	Interval     *int          `json:"interval,omitempty"`
	Schedule     *BillSchedule `json:"schedule,omitempty"`
	Name         string        `json:"name" binding:"max=19"`
	Recipient    string        `json:"recipient" binding:"required,max=255"`
	Description  string        `json:"description,max=500"`
	Reminder     *bool         `json:"reminder,omitempty"`
}

// BillUpdateRequest represents the full replacement payload for updating a bill.
type BillUpdateRequest = BillCreateRequest

// BillResponse is returned by bill endpoints with computed due-date telemetry.
type BillResponse struct {
	ID           string        `json:"id"`
	UserID       string        `json:"user_id"`
	AccountID    *string       `json:"account_id"`
	CategoryID   *string       `json:"category_id"`
	Amount       float64       `json:"amount"`
	LastDatePaid *string       `json:"last_date_paid"`
	Frequency    string        `json:"frequency"`
	Interval     int           `json:"interval"`
	Schedule     *BillSchedule `json:"schedule"`
	Name         string        `json:"name"`
	Recipient    string        `json:"recipient"`
	Description  string        `json:"description"`
	Reminder     bool          `json:"reminder"`
	NextDueDate  *string       `json:"next_due_date"`
	DueStatus    string        `json:"due_status"`
	DaysUntilDue *int          `json:"days_until_due"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// BillApplyRequest represents optional parameters when applying a bill to create a transaction record.
type BillApplyRequest struct {
	AccountID   *string `json:"account_id,omitempty"`
	Date        *string `json:"date,omitempty"`
	DateCleared *string `json:"date_cleared,omitempty"`
}

// BillApplyResponse returns both the updated bill and the newly created checking record.
type BillApplyResponse struct {
	Bill   BillResponse           `json:"bill"`
	Record CheckingRecordResponse `json:"record"`
}
