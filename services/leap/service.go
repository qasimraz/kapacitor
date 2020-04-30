package leap

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/influxdata/kapacitor/alert"
	"github.com/influxdata/kapacitor/keyvalue"
)

type Diagnostic interface {
	WithContext(ctx ...keyvalue.T) Diagnostic
	Error(msg string, err error)
}

type Service struct {
	mu sync.RWMutex
	// workspaces map[string]*Workspace
	configValue atomic.Value
	diag        Diagnostic
}

type Workspace struct {
	mu     sync.RWMutex
	config Config
	client *http.Client
}

func NewWorkspace(c Config) (*Workspace, error) {
	cl := &http.Client{}

	return &Workspace{
		config: c,
		client: cl,
	}, nil
}

func NewService(c Config, d Diagnostic) (*Service, error) {
	s := &Service{
		diag: d,
	}
	s.configValue.Store(c)

	return s, nil
}

func (s *Service) Open() error {
	return nil
}

func (s *Service) Close() error {
	return nil
}

func (s *Service) Update(newConfig []interface{}) error {
	if l := len(newConfig); l != 1 {
		return fmt.Errorf("expected only one new config object, got %d", l)
	}
	if c, ok := newConfig[0].(Config); !ok {
		return fmt.Errorf("expected config object to be of type %T, got %T", c, newConfig[0])
	} else {
		s.configValue.Store(c)
	}
	return nil
}

func (s *Service) config() Config {
	return s.configValue.Load().(Config)
}

// Alert sends an action to the specified workflow.
func (s *Service) Alert(workflow, action string) error {
	c := s.config()
	if !c.Enabled {
		return errors.New("service is not enabled")
	}
	r, err := http.Get(c.URL)
	if err != nil {
		return err
	}
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code %d from Foo service", r.StatusCode)
	}
	return nil
}

type HandlerConfig struct {
	//Workflow specifies the target workflow for the actions
	Workflow string `mapstructure:"wfe"`
}

type handler struct {
	s    *Service
	c    HandlerConfig
	diag Diagnostic
}

func (s *Service) Handler(c HandlerConfig, ctx ...keyvalue.T) (alert.Handler, error) {
	return &handler{
		s:    s,
		c:    c,
		diag: s.diag.WithContext(ctx...),
	}, nil
}

// Handle takes an event and posts its message to the Foo service chat room.
func (h *handler) Handle(event alert.Event) {
	if err := h.s.Alert(h.c.Workflow, event.State.Message); err != nil {
		h.diag.Error("failed to send event", err)
	}
}

func (s *Service) Test(options interface{}) error {
	return nil
}

func (s *Service) TestOptions() interface{} {
	return &testOptions{
		Message: "test slack message",
		Level:   alert.Critical,
	}
}

type testOptions struct {
	Workspace string      `json:"workspace"`
	Message   string      `json:"message"`
	Level     alert.Level `json:"level"`
}
