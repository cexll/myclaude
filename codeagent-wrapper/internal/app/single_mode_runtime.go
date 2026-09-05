package wrapper

import (
	"fmt"
	"reflect"

	config "codeagent-wrapper/internal/config"
	backend "codeagent-wrapper/internal/infrastructure/backend"
)

func configureSingleModeBackend(cfg *config.Config) (backend.Backend, error) {
	selected, err := selectBackendFn(cfg.Backend)
	if err != nil {
		return nil, err
	}
	cfg.Backend = selected.Name()

	cmdInjected := codexCommand != defaultCodexCommand
	argsInjected := buildCodexArgsFn != nil && reflect.ValueOf(buildCodexArgsFn).Pointer() != reflect.ValueOf(defaultBuildArgsFn).Pointer()

	if selected.Name() != defaultBackendName || !cmdInjected {
		codexCommand = selected.Command()
	}
	if selected.Name() != defaultBackendName || !argsInjected {
		buildCodexArgsFn = selected.BuildArgs
	}

	logInfo(fmt.Sprintf("Selected backend: %s", selected.Name()))
	return selected, nil
}

func requireSingleModeLogger() (*Logger, error) {
	logger := activeLogger()
	if logger == nil {
		return nil, fmt.Errorf("logger is not initialized")
	}
	return logger, nil
}
