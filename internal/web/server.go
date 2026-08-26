package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"ovencheck/internal/core"
	"ovencheck/internal/measurements"
	"ovencheck/internal/review"
	"ovencheck/internal/validation"
	"strings"
	"time"
)

//go:embed assets/index.html assets/app.css assets/app.js
var assets embed.FS

type Server struct {
	Store  *core.Store
	Review *review.Service
	mux    *http.ServeMux
}

func NewServer(store *core.Store) *Server {
	s := &Server{Store: store, Review: review.New(store), mux: http.NewServeMux()}
	s.routes()
	return s
}
func (s *Server) routes() {
	s.mux.HandleFunc("/", s.index)
	s.mux.HandleFunc("/static/", s.static)
	s.mux.HandleFunc("/api/batches", s.batches)
	s.mux.HandleFunc("/api/batches/", s.batchAction)
	s.mux.HandleFunc("/api/certificates/", s.certificate)
}
func (s *Server) Handler() http.Handler { return logging(s.mux) }
func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	b, _ := assets.ReadFile("assets/index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(b)
}
func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/static/")
	b, err := assets.ReadFile("assets/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(name, ".css") {
		w.Header().Set("Content-Type", "text/css")
	} else {
		w.Header().Set("Content-Type", "application/javascript")
	}
	w.Write(b)
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
func errorJSON(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	writeJSON(w, map[string]string{"error": err.Error(), "code": "invalid_request"})
}
func notFoundJSON(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusNotFound)
	writeJSON(w, map[string]string{"error": err.Error(), "code": "not_found"})
}
func decode(r *http.Request, v any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func (s *Server) batches(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		keyword := r.URL.Query().Get("q")
		if keyword == "" {
			keyword = r.URL.Query().Get("keyword")
		}
		status := r.URL.Query().Get("status")
		status = core.CanonicalStatus(status)
		if !core.ValidStatus(status) {
			errorJSON(w, fmt.Errorf("未知状态 %s", status))
			return
		}
		items, err := s.Store.SearchChecked(keyword, status)
		if err != nil {
			errorJSON(w, err)
			return
		}
		writeJSON(w, map[string]any{"items": items, "counts": s.Store.Counts()})
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct{ Name, KilnCode, LiningMaterial, Operator, Engineer string }
	if err := decode(r, &in); err != nil {
		errorJSON(w, err)
		return
	}
	if in.Name == "" || in.KilnCode == "" {
		errorJSON(w, fmt.Errorf("批次名称和炉号不能为空"))
		return
	}
	b, err := s.Store.Create(in.Name, in.KilnCode, in.LiningMaterial, in.Operator, in.Engineer)
	if err != nil {
		errorJSON(w, err)
		return
	}
	w.WriteHeader(201)
	writeJSON(w, b)
}
func (s *Server) batchAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/batches/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		b, ok := s.Store.Get(id)
		if !ok {
			notFoundJSON(w, fmt.Errorf("批次不存在"))
			return
		}
		writeJSON(w, b)
		return
	}
	if len(parts) == 2 && parts[1] == "report" {
		b, ok := s.Store.Get(id)
		if !ok {
			notFoundJSON(w, fmt.Errorf("批次不存在"))
			return
		}
		writeJSON(w, validation.Evaluate(b))
		return
	}
	if len(parts) == 2 && parts[1] == "evidence" && r.Method == http.MethodGet {
		b, ok := s.Store.Get(id)
		if !ok {
			notFoundJSON(w, fmt.Errorf("批次不存在"))
			return
		}
		evidence := s.Review.Evidence(b)
		evidence.Reviewer = r.URL.Query().Get("reviewer")
		evidence.Decision = r.URL.Query().Get("decision")
		evidence.Comment = review.NormalizeComment(r.URL.Query().Get("comment"))
		writeJSON(w, evidence)
		return
	}
	if len(parts) == 2 && parts[1] == "matrix" && r.Method == http.MethodGet {
		b, ok := s.Store.Get(id)
		if !ok {
			notFoundJSON(w, fmt.Errorf("批次不存在"))
			return
		}
		if b.FrozenAt == nil {
			errorJSON(w, fmt.Errorf("批次计划尚未冻结"))
			return
		}
		writeJSON(w, validation.BuildMatrix(b))
		return
	}
	if (len(parts) == 3 && parts[1] == "stages" && parts[2] != "") || (len(parts) == 4 && parts[1] == "stages" && (parts[3] == "stats" || parts[3] == "statistics")) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		b, ok := s.Store.Get(id)
		if !ok {
			notFoundJSON(w, fmt.Errorf("批次不存在"))
			return
		}
		if b.FrozenAt == nil {
			errorJSON(w, fmt.Errorf("批次计划尚未冻结"))
			return
		}
		st, ok := s.Store.Stage(id, parts[2])
		if !ok {
			http.NotFound(w, r)
			return
		}
		rs := measurements.StageReadings(b, st.ID)
		writeJSON(w, map[string]any{"stage": st, "summary": measurements.Summarize(rs), "rates": measurements.Rates(rs), "gaps": measurements.Gaps(rs, 60), "holdWindows": validation.HoldWindows(st, rs), "details": validation.DetailedStage(b, st)})
		return
	}
	if (len(parts) != 2 && !(len(parts) == 3 && parts[1] == "readings" && (parts[2] == "precheck" || parts[2] == "preview"))) || r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	action := parts[1]
	if len(parts) == 3 {
		action = parts[1] + "/" + parts[2]
	}
	switch action {
	case "stages":
		var in struct {
			ExpectedVersion int                 `json:"expectedVersion"`
			Stages          []core.HeatingStage `json:"stages"`
		}
		if err := decode(r, &in); err != nil {
			errorJSON(w, err)
			return
		}
		b, ok := s.Store.Get(id)
		if !ok {
			errorJSON(w, fmt.Errorf("批次不存在"))
			return
		}
		if err := s.Store.AddStages(id, in.ExpectedVersion, in.Stages); err != nil {
			errorJSON(w, err)
			return
		}
		b, _ = s.Store.Get(id)
		writeJSON(w, map[string]any{"batch": b, "stages": core.OrderedStages(b), "planVersion": b.PlanVersion})
	case "freeze":
		var in struct {
			ExpectedVersion int `json:"expectedVersion"`
		}
		if err := decode(r, &in); err != nil {
			errorJSON(w, err)
			return
		}
		if err := s.Store.Freeze(id, in.ExpectedVersion); err != nil {
			errorJSON(w, err)
			return
		}
		b, _ := s.Store.Get(id)
		writeJSON(w, b)
	case "readings":
		var in struct {
			ExpectedVersion int                  `json:"expectedVersion"`
			Readings        []measurements.Input `json:"readings"`
		}
		if err := decode(r, &in); err != nil {
			errorJSON(w, err)
			return
		}
		b, ok := s.Store.Get(id)
		if !ok {
			errorJSON(w, fmt.Errorf("批次不存在"))
			return
		}
		rs, err := measurements.Normalize(b, in.Readings)
		if err == nil {
			err = measurements.ValidateOrder(b.Readings, rs)
		}
		if err == nil {
			err = measurements.CheckTimestampRange(rs, 0, time.Now())
		}
		if err == nil {
			err = s.Store.AddReadings(id, in.ExpectedVersion, rs)
		}
		if err != nil {
			errorJSON(w, err)
			return
		}
		b, _ = s.Store.Get(id)
		_ = s.Review.RefreshRetests(id)
		b, _ = s.Store.Get(id)
		writeJSON(w, b)
	case "readings/precheck", "readings/preview":
		b, ok := s.Store.Get(id)
		if !ok {
			errorJSON(w, fmt.Errorf("批次不存在"))
			return
		}
		inputs, err := parseImportRequest(r)
		if err != nil {
			errorJSON(w, err)
			return
		}
		result := measurements.Import(b, inputs)
		if result.Accepted > 0 {
			if err := measurements.CheckTimestampRange(result.Readings, 0, time.Now()); err != nil {
				result.Accepted = 0
				result.Rejected = len(inputs)
				result.Readings = nil
				result.Errors = append(result.Errors, err.Error())
			}
		}
		writeJSON(w, map[string]any{"batchVersion": b.Version, "result": result})
	case "actions":
		var in struct {
			ExpectedVersion                                int `json:"expectedVersion"`
			StageID, Kind, Reason, ActionText, PerformedBy string
			RetestRequired                                 bool   `json:"retestRequired"`
			PerformedAt                                    string `json:"performedAt"`
		}
		if err := decode(r, &in); err != nil {
			errorJSON(w, err)
			return
		}
		performedAt := time.Time{}
		if in.PerformedAt != "" {
			var err error
			performedAt, err = time.Parse(time.RFC3339, in.PerformedAt)
			if err != nil {
				errorJSON(w, fmt.Errorf("处置时间格式错误"))
				return
			}
		}
		if err := s.Review.RecordDeviationAt(id, in.ExpectedVersion, in.StageID, in.Kind, in.Reason, in.ActionText, in.PerformedBy, in.RetestRequired, performedAt); err != nil {
			errorJSON(w, err)
			return
		}
		b, _ := s.Store.Get(id)
		writeJSON(w, b)
	case "review":
		var in struct {
			ExpectedVersion             int `json:"expectedVersion"`
			Reviewer, Decision, Comment string
		}
		if err := decode(r, &in); err != nil {
			errorJSON(w, err)
			return
		}
		c, err := s.Review.Decide(id, in.ExpectedVersion, in.Reviewer, in.Decision, in.Comment)
		if err != nil {
			errorJSON(w, err)
			return
		}
		writeJSON(w, c)
	default:
		http.NotFound(w, r)
	}
}

func parseImportRequest(r *http.Request) ([]measurements.Input, error) {
	content := r.Header.Get("Content-Type")
	if strings.HasPrefix(strings.ToLower(content), "text/csv") {
		return measurements.ParseCSV(r.Body)
	}
	var in struct {
		Readings []measurements.Input `json:"readings"`
		CSV      string               `json:"csv"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	if in.CSV != "" {
		return measurements.ParseCSV(strings.NewReader(in.CSV))
	}
	return in.Readings, nil
}
func (s *Server) certificate(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/certificates/")
	if strings.HasSuffix(id, "/verify") {
		id = strings.TrimSuffix(id, "/verify")
		for _, b := range s.Store.List() {
			if b.Certificate != nil && b.Certificate.ID == id {
				writeJSON(w, map[string]any{"certificateID": id, "valid": review.VerifyCertificate(*b.Certificate), "batchVersion": b.Certificate.BatchVersion})
				return
			}
		}
		http.NotFound(w, r)
		return
	}
	for _, b := range s.Store.List() {
		if b.Certificate != nil && b.Certificate.ID == id {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename=oven-certificate.txt")
			w.Write([]byte(review.CertificateText(*b.Certificate)))
			return
		}
	}
	http.NotFound(w, r)
}
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Time", time.Now().Format(time.RFC3339))
		next.ServeHTTP(w, r)
	})
}
