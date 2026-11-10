package midtrans

import "context"

var _ Client = (*MockClient)(nil)

type MockClient struct {
	CreateTransactionFunc func(ctx context.Context, req TransactionRequest) (TransactionResponse, error)
	GetStatusFunc         func(ctx context.Context, orderID string) (StatusResponse, error)
	ExpireTransactionFunc func(ctx context.Context, orderID string) error
}

func (m *MockClient) CreateTransaction(ctx context.Context, req TransactionRequest) (TransactionResponse, error) {
	return m.CreateTransactionFunc(ctx, req)
}

func (m *MockClient) GetStatus(ctx context.Context, orderID string) (StatusResponse, error) {
	return m.GetStatusFunc(ctx, orderID)
}

func (m *MockClient) ExpireTransaction(ctx context.Context, orderID string) error {
	return m.ExpireTransactionFunc(ctx, orderID)
}
