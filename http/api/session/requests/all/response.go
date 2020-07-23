package all

type record struct {
	ClientAddr    string            `json:"client_address"`
	Method        string            `json:"method"`
	Content       string            `json:"content"`
	Headers       map[string]string `json:"headers"`
	URI           string            `json:"url"`
	CreatedAtUnix int64             `json:"created_at_unix"`
}

type response map[string]record
