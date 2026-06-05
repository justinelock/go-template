package vo

// CreateOrderResp 创建订单响应。
type CreateOrderResp struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
}

// OrderResp 订单详情响应。
type OrderResp struct {
	OrderID        string  `json:"orderId"`
	UserID         string  `json:"userId"`
	ProductID      string  `json:"productId"`
	Amount         float64 `json:"amount"`
	Status         string  `json:"status"`
	IdempotencyKey string  `json:"idempotencyKey,omitempty"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt,omitempty"`
}

// FromDomain 将领域字段映射为对外 OrderResp。
func FromDomain(orderID, userID, productID string, amount float64, status, idempotencyKey, createdAt, updatedAt string) OrderResp {
	// 步骤 1：逐字段组装响应 VO。
	return OrderResp{
		OrderID:        orderID,
		UserID:         userID,
		ProductID:      productID,
		Amount:         amount,
		Status:         status,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}
