package connections

// ActionTarget identifies the connection an action acts on. Instance and PID
// describe the selected row; ConnectionID, TransactionID, and QueryID are the
// server-issued IDs for connection, transaction, and query verification.
type ActionTarget struct {
	Instance      string
	PID           int
	ConnectionID  *string
	TransactionID *string
	QueryID       *string
}
