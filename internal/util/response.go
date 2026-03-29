package util

type Envelope struct {
	OK        bool         `json:"ok"`
	RequestID string       `json:"request_id,omitempty"`
	Data      any          `json:"data,omitempty"`
	Error     *ErrorDetail `json:"error,omitempty"`
}

func Ok(data any, requestID string) Envelope {
	return Envelope{OK: true, Data: data, RequestID: requestID}
}

func Fail(err *ErrorDetail, requestID string) Envelope {
	return Envelope{OK: false, Error: err, RequestID: requestID}
}
