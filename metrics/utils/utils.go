// Copyright 2024 Harness Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utils

import "time"

func CalculateDuration(startTime time.Time) float64 {
	taskExecutionTime := time.Since(startTime).Seconds()
	return taskExecutionTime
}
