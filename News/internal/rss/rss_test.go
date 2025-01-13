package rss

import (
	"testing"
)

func Test_getRSS(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "RSS_url_1",
			args: args{
				url: "https://habr.com/ru/rss/hub/go/all/?fl=ru",
			},
			wantErr: false,
		},
		{
			name: "RSS_url_2",
			args: args{
				url: "https://habr.com/ru/rss/best/daily/?fl=ru",
			},
			wantErr: false,
		},
		{
			name: "RSS_url_3",
			args: args{
				url: "https://cprss.s3.amazonaws.com/golangweekly.com.xml",
			},
			wantErr: false,
		},
		{
			name: "RSS_url_4_error",
			args: args{
				url: "https://localhost:1111/",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := GetRSS(tt.args.url)

			if (err != nil) != tt.wantErr {
				t.Errorf("getRSS() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err != nil {
				t.Log(err)
				return
			}

			if (err == nil) && (got == nil) {
				t.Errorf("getRSS() got = nil, want not nil")
				return
			} else if got != nil {
				t.Log("Последняя RSS: ", got[0].Title)
			}
		})
	}
}
