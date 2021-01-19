// Fake storage, just for a test

package null

import st "github.com/tarampampam/webhook-tester/internal/pkg/storage"

type Storage struct {
	SessionData *st.SessionData
	RequestData *st.RequestData
	Requests    *[]st.RequestData
	Error       error
	Boolean     bool
}

func (s *Storage) DeleteSession(_ string) (bool, error)               { return s.Boolean, s.Error }
func (s *Storage) DeleteRequests(_ string) (bool, error)              { return s.Boolean, s.Error }
func (s *Storage) DeleteRequest(_, _ string) (bool, error)            { return s.Boolean, s.Error }
func (s *Storage) GetSession(_ string) (*st.SessionData, error)       { return s.SessionData, s.Error }
func (s *Storage) GetRequest(_, _ string) (*st.RequestData, error)    { return s.RequestData, s.Error }
func (s *Storage) GetAllRequests(_ string) (*[]st.RequestData, error) { return s.Requests, s.Error }
func (s *Storage) CreateSession(_ *st.WebHookResponse) (*st.SessionData, error) {
	return s.SessionData, s.Error
}
func (s *Storage) CreateRequest(_ string, _ *st.Request) (*st.RequestData, error) {
	return s.RequestData, s.Error
}
