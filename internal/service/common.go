package service

import (
	"time"

	apiv1 "github.com/squall-chua/go-support-ticket/api/v1"
)

// getPaginationParams calculates limit, offset, and pageNumber from a Pagination object.
// Default limit is 10 if not provided.
func getPaginationParams(p *apiv1.PageRequest) (int32, int32, int32) {
	var limit, offset, pageNumber int32 = 10, 0, 1
	if p != nil {
		if p.PageSize > 0 {
			limit = p.PageSize
		}
		if p.PageNumber > 1 {
			pageNumber = p.PageNumber
			offset = (pageNumber - 1) * limit
		}
	}
	return limit, offset, pageNumber
}

// getTimeRange extracts start and end times from a TimeRange proto object.
func getTimeRange(tr *apiv1.TimeRange) (*time.Time, *time.Time) {
	if tr == nil {
		return nil, nil
	}
	var start, end *time.Time
	if tr.StartTime != nil {
		t := tr.StartTime.AsTime()
		start = &t
	}
	if tr.EndTime != nil {
		t := tr.EndTime.AsTime()
		end = &t
	}
	return start, end
}
