package cleanup

import (
	"fmt"
	"io"

	logger "codeagent-wrapper/internal/logger"
)

type Deps struct {
	Run func() (logger.CleanupStats, error)
}

func Execute(deps Deps) (logger.CleanupStats, error) {
	if deps.Run == nil {
		return logger.CleanupStats{}, fmt.Errorf("log cleanup function not configured")
	}
	return deps.Run()
}

func Render(w io.Writer, stats logger.CleanupStats) {
	if w == nil {
		return
	}

	fmt.Fprintln(w, "Cleanup completed")
	fmt.Fprintf(w, "Files scanned: %d\n", stats.Scanned)
	fmt.Fprintf(w, "Files deleted: %d\n", stats.Deleted)
	if len(stats.DeletedFiles) > 0 {
		for _, f := range stats.DeletedFiles {
			fmt.Fprintf(w, "  - %s\n", f)
		}
	}
	fmt.Fprintf(w, "Files kept: %d\n", stats.Kept)
	if len(stats.KeptFiles) > 0 {
		for _, f := range stats.KeptFiles {
			fmt.Fprintf(w, "  - %s\n", f)
		}
	}
	if stats.Errors > 0 {
		fmt.Fprintf(w, "Deletion errors: %d\n", stats.Errors)
	}
}
