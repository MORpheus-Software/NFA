type OpenSessionWithFailover struct {
    SessionDuration *lib.BigInt `json:"sessionDuration" binding:"required" validate:"required"`
    DirectPayment   bool        `json:"directPayment" binding:"omitempty"`
    Failover        bool        `json:"failover" binding:"omitempty"`
} 