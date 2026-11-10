package midtrans

import "context"

type TransactionDetails struct {
	OrderID     string `json:"order_id"`
	GrossAmount int64  `json:"gross_amount"`
}

type ItemDetail struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Price    int64  `json:"price"`
	Quantity int    `json:"quantity"`
}

type CustomerDetails struct {
	FirstName string `json:"first_name"`
	Phone     string `json:"phone"`
}

type Expiry struct {
	StartTime string `json:"start_time"`
	Unit      string `json:"unit"`
	Duration  int    `json:"duration"`
}

type Callbacks struct {
	Finish string `json:"finish"`
}

type TransactionRequest struct {
	TransactionDetails TransactionDetails `json:"transaction_details"`
	ItemDetails        []ItemDetail       `json:"item_details"`
	CustomerDetails    CustomerDetails    `json:"customer_details"`
	Expiry             *Expiry            `json:"expiry,omitempty"`
	Callbacks          *Callbacks         `json:"callbacks,omitempty"`
}

type TransactionResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

type StatusResponse struct {
	OrderID           string `json:"order_id"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	GrossAmount       string `json:"gross_amount"`
	PaymentType       string `json:"payment_type"`
	TransactionID     string `json:"transaction_id"`
}

type Client interface {
	CreateTransaction(ctx context.Context, req TransactionRequest) (TransactionResponse, error)
	GetStatus(ctx context.Context, orderID string) (StatusResponse, error)
	ExpireTransaction(ctx context.Context, orderID string) error
}
