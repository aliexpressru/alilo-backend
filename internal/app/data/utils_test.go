package data

import "testing"

func TestCalculateTotalPages(t *testing.T) {
	type args struct {
		numberLines int64
		limit       int32
	}
	tests := []struct {
		name           string
		args           args
		wantTotalPages int64
	}{
		{
			name: "10/0",
			args: args{
				numberLines: 0,
				limit:       10,
			},
			wantTotalPages: 0,
		},
		{
			name: "1/10",
			args: args{
				numberLines: 10,
				limit:       1,
			},
			wantTotalPages: 10,
		},
		{
			name: "9/10",
			args: args{
				numberLines: 10,
				limit:       9,
			},
			wantTotalPages: 2,
		},
		{
			name: "10/10",
			args: args{
				numberLines: 10,
				limit:       10,
			},
			wantTotalPages: 1,
		},
		{
			name: "11/10",
			args: args{
				numberLines: 10,
				limit:       11,
			},
			wantTotalPages: 1,
		},
		{
			name: "2/0",
			args: args{
				numberLines: 0,
				limit:       2,
			},
			wantTotalPages: 0,
		},
		{
			name: "1/100",
			args: args{
				numberLines: 5,
				limit:       100,
			},
			wantTotalPages: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotTotalPages := CalculateTotalPages(tt.args.numberLines, tt.args.limit); gotTotalPages != tt.wantTotalPages {
				t.Errorf("got TotalPages = %v, want %v", gotTotalPages, tt.wantTotalPages)
			}
		})
	}
}

func TestOffsetCalculation(t *testing.T) {
	type args struct {
		limit      int32
		pageNumber int32
	}
	tests := []struct {
		name            string
		args            args
		wantOffset      int32
		wantReturnLimit int32
	}{
		{
			name: "Zero",
			args: args{
				limit:      0,
				pageNumber: 0,
			},
			wantOffset:      0,
			wantReturnLimit: 10,
		},
		{
			name: "10/0",
			args: args{
				limit:      0,
				pageNumber: 10,
			},
			wantOffset:      90,
			wantReturnLimit: 10,
		},
		{
			name: "0/10",
			args: args{
				limit:      10,
				pageNumber: 0,
			},
			wantOffset:      0,
			wantReturnLimit: 10,
		},
		{
			name: "1/10",
			args: args{
				limit:      10,
				pageNumber: 1,
			},
			wantOffset:      0,
			wantReturnLimit: 10,
		},
		{
			name: "9/10",
			args: args{
				limit:      10,
				pageNumber: 9,
			},
			wantOffset:      80,
			wantReturnLimit: 10,
		},
		{
			name: "10/10",
			args: args{
				limit:      10,
				pageNumber: 10,
			},
			wantOffset:      90,
			wantReturnLimit: 10,
		},
		{
			name: "11/10",
			args: args{
				limit:      10,
				pageNumber: 11,
			},
			wantOffset:      100,
			wantReturnLimit: 10,
		},
		{
			name: "2/2",
			args: args{
				limit:      2,
				pageNumber: 2,
			},
			wantOffset:      2,
			wantReturnLimit: 2,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOffset, gotReturnLimit := OffsetCalculation(tt.args.limit, tt.args.pageNumber)
			if gotOffset != tt.wantOffset {
				t.Errorf("OffsetCalculation() got Offset = %v, want %v", gotOffset, tt.wantOffset)
			}
			if gotReturnLimit != tt.wantReturnLimit {
				t.Errorf("OffsetCalculation() got ReturnLimit = %v, want %v", gotReturnLimit, tt.wantReturnLimit)
			}
		})
	}
}
