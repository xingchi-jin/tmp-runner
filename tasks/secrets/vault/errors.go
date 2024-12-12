package vault

func NewErrorResponse(err error, message string, status int) ErrorResponse {
	return ErrorResponse{
		Message: message,
		Error:   err.Error(),
		Status:  status,
	}
}
