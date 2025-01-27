package utils

// Function to check if a string pointer is nil or points to an empty string
func IsNilOrEmpty(s *string) bool {
	// Check if the string pointer is nil or if the value it points to is empty
	return s == nil || *s == ""
}

// SafeStringAssertion is a helper function that safely performs type assertion for a string value
// It returns the value as a string if it exists, or the defaultValue provided
func SafeStringAssertion(data map[string]interface{}, key string, defaultValue string) string {
	if value, exists := data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// SafeBoolAssertion is a helper function that safely performs type assertion for a bool value
// It returns the value as a bool if it exists, or the defaultValue provided
func SafeBoolAssertion(data map[string]interface{}, key string, defaultValue bool) bool {
	if value, exists := data[key]; exists {
		if boolean, ok := value.(bool); ok {
			return boolean
		}
	}
	return defaultValue
}
