package api

import "sort"

type StatusResponse struct {
	Success bool `json:"success"`
}

type (
	RequestHeader struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	StoredRequest struct {
		UUID          string         `json:"uuid"`
		ClientAddr    string         `json:"client_address"`
		Method        string         `json:"method"`
		Content       string         `json:"content"`
		Headers       RequestHeaders `json:"headers"`
		URI           string         `json:"url"`
		CreatedAtUnix int64          `json:"created_at_unix"`
	}
)

type StoredRequests []StoredRequest

func (r StoredRequests) Sorted() StoredRequests {
	sort.SliceStable(r, func(i, j int) bool {
		return r[i].CreatedAtUnix > r[j].CreatedAtUnix
	})

	return r
}

type RequestHeaders []RequestHeader

func (h RequestHeaders) Sorted() RequestHeaders {
	sort.SliceStable(h, func(i, j int) bool {
		return h[i].Name < h[j].Name
	})

	return h
}

func MapToHeaders(in map[string]string) *RequestHeaders {
	result := make(RequestHeaders, 0, len(in))

	for name, value := range in {
		result = append(result, RequestHeader{Name: name, Value: value})
	}

	return &result
}
