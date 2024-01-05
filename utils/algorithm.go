package utils

import (
	"fmt"
	"time"
)

var EbbinghausIntervals = []int{1, 1, 2, 3, 5, 7, 9, 11, 13, 15}

func GetNextReviewDate(today time.Time, reviewsCompleted int) time.Time {
	if reviewsCompleted < 0 || reviewsCompleted >= len(EbbinghausIntervals) {
		fmt.Println("Error: reviewsCompleted is out of the valid range")
		return today
	}

	interval := EbbinghausIntervals[reviewsCompleted]
	nextReviewDate := today.AddDate(0, 0, interval)
	return nextReviewDate
}
