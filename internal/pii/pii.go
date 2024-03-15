package pii

type Detector interface {
	Detect(input []string) (*Result, error)
}

type Scanner struct {
	detector Detector
}

type Detection struct {
	Input    string
	Entities []*Entity
}

type Entity struct {
	BeginOffset int
	EndOffset   int
	Type        string
}

type Result struct {
	Detections []*Detection
}

func NewScanner(d Detector) *Scanner {
	return &Scanner{
		detector: d,
	}
}

func (s *Scanner) Scan(input []string) (*Result, error) {
	return s.detector.Detect(
		input,
	)
}
