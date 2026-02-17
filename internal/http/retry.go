package apixhttp

import "time"

func shouldRetryStatus(statusCode int) bool {
	return statusCode >= 500 && statusCode <= 599
}

func shouldRetryNetworkError(err error) bool {
	return err != nil
}

func retryDelayForAttempt(base time.Duration, attempt int) time.Duration {
	if base <= 0 {
		return 0
	}
	if attempt <= 1 {
		return base
	}

	delay := base
	for i := 1; i < attempt; i++ {
		delay *= 2
	}
	return delay
}
