package wrapper

import backend "codeagent-wrapper/internal/backend"

func selectBackend(name string) (Backend, error) { return backend.Select(name) }
