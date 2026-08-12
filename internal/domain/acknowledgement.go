package domain

type AckAction string

const (
	AckActionAck      AckAction = "ack"
	AckActionReject   AckAction = "reject"
	AckActionOverride AckAction = "override"
)

type Acknowledgement struct {
	ID          string
	DetectionID string
	Action      AckAction
}
