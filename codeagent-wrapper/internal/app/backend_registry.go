package wrapper

import backend "codeagent-wrapper/internal/infrastructure/backend"

func selectBackend(name string) (backend.Backend, error) { return backend.Select(name) }
