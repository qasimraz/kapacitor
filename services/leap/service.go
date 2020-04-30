package leap

import (
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/influxdata/kapacitor/alert"
	"github.com/influxdata/kapacitor/keyvalue"
)

type Diagnostic interface {
	WithContext(ctx ...keyvalue.T) Diagnostic
	Error(msg string, err error)
}

type Service struct {
	mu         sync.RWMutex
	workspaces map[string]*Workspace
	// config     Config
	diag Diagnostic
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
		diag:       d,
		workspaces: make(map[string]*Workspace),
	}

	w, err := NewWorkspace(c)
	if err != nil {
		return nil, err
	}
	s.workspaces[c.Workspace] = w

	// We'll stash the default workspace with the empty string as a key.
	// Either there's a single config with no workspace name, or else
	// we have multiple configs that all have names.
	if c.Workspace != "" {
		s.workspaces[""] = s.workspaces[c.Workspace]
	}

	return s, nil
}

func (s *Service) Open() error {
	// Perform any initialization needed here
	return nil
}

func (s *Service) Close() error {
	// Perform any actions needed to properly close the service here.
	// For example signal and wait for all go routines to finish.
	return nil
}

func (s *Service) Update(newConfigs []interface{}) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, v := range newConfigs {
		if conf, ok := v.(Config); ok {
			_, ok := s.workspaces[conf.Workspace]
			if !ok {
				w, err := NewWorkspace(conf)
				s.workspaces[conf.Workspace] = w
				if err != nil {
					return err
				}
			}

			// We'll stash the default workspace with the empty string as a key.
			// Either there's a single config with no workspace name, or else
			// we have multiple configs that all have names.
			if conf.Workspace != "" {
				s.workspaces[""] = s.workspaces[conf.Workspace]
			}
		} else {
			return fmt.Errorf("expected config object to be of type %T, got %T", v, conf)
		}
	}
	return nil
}

// config loads the config struct stored in the configValue field.
// func (s *Service) config() Config {
// 	return s.Work.Load().(Config)
// }

func (s *Service) config(wid string) (Config, error) {
	w, err := s.workspace(wid)
	if err != nil {
		return Config{}, err
	}

	return w.Config(), err
}

func (w *Workspace) Config() Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.config
}

func (s *Service) workspace(wid string) (*Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.workspaces) == 0 {
		return &Workspace{}, errors.New("no slack configuration found")
	}
	v, ok := s.workspaces[wid]
	if !ok {
		return &Workspace{}, errors.New("workspace id not found")
	}
	return v, nil
}

// Alert sends an action to the specified workflow.
func (s *Service) Alert(workflow, action string) error {
	c, err := s.config("URL")
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

// handler provides the implementation of the alert.Handler interface for the Foo service.
// type handler struct {
// 	s      *Service
// 	c      HandlerConfig
// 	logger *log.Logger
// }

// // DefaultHandlerConfig returns a HandlerConfig struct with defaults applied.
// func (s *Service) DefaultHandlerConfig() HandlerConfig {
// 	// return a handler config populated with the default workflow from the service config.
// 	c := s.config()
// 	return HandlerConfig{
// 		Workflow: c.Workflow,
// 	}
// }

// Handler creates a handler from the config.
// func (s *Service) Handler(c HandlerConfig, l *log.Logger) alert.Handler {
// 	// handlers can operate in differing contexts, as a result a logger is passed
// 	// in so that logs from this handler can be correctly associatied with a given context.
// 	return &handler{
// 		s:      s,
// 		c:      c,
// 		logger: l,
// 	}
// }

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
	// o, ok := options.(*testOptions)
	// if !ok {
	// 	return fmt.Errorf("unexpected options type %T", options)
	// }
	// return s.Alert(o.Workspace, o.Channel, o.Message, o.Username, o.IconEmoji, o.Level)
	return nil
}

func (s *Service) TestOptions() interface{} {
	c, _ := s.config("")
	return &testOptions{
		Workspace: c.Workspace,
		Message:   "test slack message",
		Level:     alert.Critical,
	}
}

type testOptions struct {
	Workspace string      `json:"workspace"`
	Message   string      `json:"message"`
	Level     alert.Level `json:"level"`
}
