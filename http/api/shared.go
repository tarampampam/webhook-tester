package api

import "sort"

type Status struct {
	Success bool `json:"success"`
}

type (
	Header struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	Request struct {
		UUID          string  `json:"uuid"`
		ClientAddr    string  `json:"client_address"`
		Method        string  `json:"method"`
		Content       string  `json:"content"`
		Headers       Headers `json:"headers"`
		URI           string  `json:"url"`
		CreatedAtUnix int64   `json:"created_at_unix"`
	}
)

type Requests []Request

func (r Requests) Sorted() Requests {
	sort.SliceStable(r, func(i, j int) bool {
		return r[i].CreatedAtUnix > r[j].CreatedAtUnix
	})

	return r
}

type Headers []Header

func (h Headers) Sorted() Headers {
	sort.SliceStable(h, func(i, j int) bool {
		return h[i].Name < h[j].Name
	})

	return h
}

func MapToHeaders(in map[string]string) *Headers {
	result := make(Headers, 0)

	for name, value := range in {
		result = append(result, Header{Name: name, Value: value})
	}

	return &result
}
