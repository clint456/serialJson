package serialcomm

type Reading struct {
	ID           string `json:"id"`
	Origin       int64  `json:"origin"`
	DeviceName   string `json:"deviceName"`
	ResourceName string `json:"resourceName"`
	ProfileName  string `json:"profileName"`
	ValueType    string `json:"valueType"`
	Value        string `json:"value"`
}

type Event struct {
	APIVersion  string    `json:"apiVersion"`
	ID          string    `json:"id"`
	DeviceName  string    `json:"deviceName"`
	ProfileName string    `json:"profileName"`
	SourceName  string    `json:"sourceName"`
	Origin      int64     `json:"origin"`
	Readings    []Reading `json:"readings"`
}

type Payload struct {
	APIVersion string `json:"apiVersion"`
	RequestID  string `json:"requestID"`
	Event      Event  `json:"event"`
}

type Message struct {
	APIVersion    string `json:"apiVersion"`
	ReceivedTopic string `json:"receivedTopic"`
	CorrelationID string `json:"correlationID"`
	RequestID     string `json:"requestID"`
	ErrorCode     int    `json:"errorCode"`
	Payload       string `json:"payload"`     // base64-encoded
	ContentType   string `json:"contentType"` // e.g. "application/json"
}
