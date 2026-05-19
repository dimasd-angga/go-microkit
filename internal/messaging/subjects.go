package messaging

// Event subjects. Keep these in one place so producers and consumers don't
// drift.
const (
	StreamOrders = "ORDERS"

	SubjOrderCreated = "orders.created"
	SubjOrderUpdated = "orders.updated"
	SubjOrderPaid    = "orders.paid"
	SubjOrderFailed  = "orders.failed"

	StreamUsers = "USERS"

	SubjUserRegistered = "users.registered"
)
