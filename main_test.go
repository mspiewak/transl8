package main

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"cloud.google.com/go/translate"
	"golang.org/x/text/language"
)

func Test_app_routeRequest(t *testing.T) {
	type source struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Type  string `json:"type"`
	}
	type want struct {
		connectivityData map[int]map[string]language.Tag
		response         string
	}
	type fields struct {
		client           *translate.Client
		connectivityData map[int]map[string]language.Tag
	}
	type args struct {
		req  reqStruct
		conf int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    want
		wantErr bool
	}{
		{
			"basic create",
			fields{nil, make(map[int]map[string]language.Tag)},
			args{
				req: reqStruct{
					Source: source{
						ID:   "forum1",
						Type: "forum",
					},
					Raw: "@Transl8 start conference en",
				},
			},
			want{
				map[int]map[string]language.Tag{
					1: {
						"f:forum1": language.English,
					},
				},
				"Created conference ID: ###",
			},
			false,
		},
		{
			"basic Join",
			fields{
				nil,
				map[int]map[string]language.Tag{
					1: {
						"ABC": language.Czech,
					},
				},
			},
			args{
				req: reqStruct{
					Source: source{
						ID:   "forum1",
						Type: "forum",
					},
					Raw: "@Transl8 join conference 1 fr",
				},
				conf: 1,
			},
			want{
				map[int]map[string]language.Tag{
					1: {
						"ABC":      language.Czech,
						"f:forum1": language.French,
					},
				},
				"Joined conference",
			},
			false,
		},
		{
			"basic Leave",
			fields{
				nil,
				map[int]map[string]language.Tag{
					1: {
						"ABC":      language.Czech,
						"f:forum1": language.French,
					},
				},
			},
			args{
				req: reqStruct{
					Source: source{
						ID:   "forum1",
						Type: "forum",
					},
					Raw: "@Transl8 leave conference",
				},
				conf: 1,
			},
			want{
				map[int]map[string]language.Tag{
					1: {
						"ABC": language.Czech,
					},
				},
				"Left conference",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &app{
				client:           tt.fields.client,
				connectivityData: tt.fields.connectivityData,
			}
			g, err := a.routeRequest(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("app.routeRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			r, err := regexp.Compile("([0-9]+)")
			if err != nil {
				t.Fatalf("Regex bad")
			}
			cID := r.FindString(g)
			if len(cID) > 0 {
				tt.args.conf, err = strconv.Atoi(strings.TrimSpace(cID))
				if err != nil {
					t.Errorf("Conference not valid")
				}
			}
			if tt.want.response != r.ReplaceAllString(g, "###") {
				t.Errorf("Wrong response: %s", r.ReplaceAllString(g, "###"))
			}

			if !reflect.DeepEqual(tt.want.connectivityData[1], a.connectivityData[tt.args.conf]) {
				t.Errorf("Conference populated wrong: %v", a.connectivityData)
			}
		})
	}
}
