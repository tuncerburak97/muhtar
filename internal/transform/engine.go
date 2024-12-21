package transform

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/dop251/goja"
	"github.com/rs/zerolog/log"
	"github.com/tuncerburak97/muhtar/internal/config"
)

// Engine handles request/response transformations
type Engine struct {
	config     config.TransformConfig
	vm         *goja.Runtime
	scripts    map[string]*goja.Program
	scriptLock sync.RWMutex
}

// NewEngine creates a new transformation engine
func NewEngine(cfg config.TransformConfig) (*Engine, error) {
	engine := &Engine{
		config:  cfg,
		vm:      goja.New(),
		scripts: make(map[string]*goja.Program),
	}

	// Load all scripts
	if err := engine.loadScripts(); err != nil {
		return nil, err
	}

	return engine, nil
}

// loadScripts loads all transformation scripts from the configured directory
func (e *Engine) loadScripts() error {
	for _, service := range e.config.Services {
		// Create script paths
		requestScriptPath := filepath.Join(e.config.ScriptsDir, service.ServiceName, "request.js")
		responseScriptPath := filepath.Join(e.config.ScriptsDir, service.ServiceName, "response.js")

		// Load request script
		requestScript, err := e.compileScript(requestScriptPath)
		if err != nil {
			return fmt.Errorf("failed to compile request script for service %s: %v", service.ServiceName, err)
		}
		e.scripts[requestScriptPath] = requestScript

		// Load response script
		responseScript, err := e.compileScript(responseScriptPath)
		if err != nil {
			return fmt.Errorf("failed to compile response script for service %s: %v", service.ServiceName, err)
		}
		e.scripts[responseScriptPath] = responseScript
	}
	return nil
}

// compileScript compiles a JavaScript file into a program
func (e *Engine) compileScript(path string) (*goja.Program, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return goja.Compile(path, string(content), true)
}

// TransformRequest transforms an HTTP request based on service configuration
func (e *Engine) TransformRequest(req *http.Request) error {
	service := e.findMatchingService(req.URL.Path)
	if service == nil {
		return nil
	}

	scriptPath := e.getScriptPath(service, true)
	script := e.scripts[scriptPath]
	if script == nil {
		return fmt.Errorf("script not found: %s", scriptPath)
	}

	// Prepare request object for script
	reqObj := map[string]interface{}{
		"method":  req.Method,
		"path":    req.URL.Path,
		"headers": headerToMap(req.Header),
	}

	// Read body if present
	if req.Body != nil {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return err
		}
		var jsonBody interface{}
		if err := json.Unmarshal(body, &jsonBody); err == nil {
			reqObj["body"] = jsonBody
		} else {
			reqObj["body"] = string(body)
		}
	}

	// Execute transformation
	vm := goja.New()
	vm.Set("request", reqObj)
	vm.Set("log", log.Logger)

	_, err := vm.RunProgram(script)
	if err != nil {
		return err
	}

	// Apply transformations back to request
	result := vm.Get("request").ToObject(vm)
	if headers := result.Get("headers"); headers != nil {
		headerMap := headers.Export().(map[string]interface{})
		for k, v := range headerMap {
			req.Header.Set(k, fmt.Sprint(v))
		}
	}

	return nil
}

// TransformResponse transforms an HTTP response based on service configuration
func (e *Engine) TransformResponse(resp *http.Response) error {
	service := e.findMatchingService(resp.Request.URL.Path)
	if service == nil {
		return nil
	}

	scriptPath := e.getScriptPath(service, false)
	script := e.scripts[scriptPath]
	if script == nil {
		return fmt.Errorf("script not found: %s", scriptPath)
	}

	// Prepare response object for script
	respObj := map[string]interface{}{
		"statusCode": resp.StatusCode,
		"headers":    headerToMap(resp.Header),
	}

	// Read body if present
	if resp.Body != nil {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var jsonBody interface{}
		if err := json.Unmarshal(body, &jsonBody); err == nil {
			respObj["body"] = jsonBody
		} else {
			respObj["body"] = string(body)
		}
	}

	// Execute transformation
	vm := goja.New()
	vm.Set("response", respObj)
	vm.Set("log", log.Logger)

	_, err := vm.RunProgram(script)
	if err != nil {
		return err
	}

	// Apply transformations back to response
	result := vm.Get("response").ToObject(vm)
	if headers := result.Get("headers"); headers != nil {
		headerMap := headers.Export().(map[string]interface{})
		for k, v := range headerMap {
			resp.Header.Set(k, fmt.Sprint(v))
		}
	}

	return nil
}

// findMatchingService finds a service configuration matching the given path
func (e *Engine) findMatchingService(path string) *config.ServiceTransform {
	for _, service := range e.config.Services {
		if service.URL == path {
			return &service
		}
	}
	return nil
}

// getScriptPath returns the appropriate script path for a service
func (e *Engine) getScriptPath(service *config.ServiceTransform, isRequest bool) string {
	scriptName := "response.js"
	if isRequest {
		scriptName = "request.js"
	}
	return filepath.Join(e.config.ScriptsDir, service.ServiceName, scriptName)
}

// headerToMap converts http.Header to map[string]interface{}
func headerToMap(header http.Header) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range header {
		if len(v) == 1 {
			result[k] = v[0]
		} else {
			result[k] = v
		}
	}
	return result
}
