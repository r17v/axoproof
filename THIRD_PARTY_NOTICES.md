# Third-Party Notices

AxoProof is built in pure Go (`CGO_ENABLED=0`) and licensed under the Apache License, Version 2.0.

## Runtime Dependencies

1. **`modernc.org/sqlite` (and transitive `modernc.org/*` packages)**
   - License: BSD-3-Clause
   - Usage: Pure-Go SQLite database driver used for counterfactual before/after database row-delta probes and optional local `.axoloom/axoloom.db` graph inspection.

2. **`github.com/google/uuid`**
   - License: BSD-3-Clause

3. **`github.com/dustin/go-humanize`**
   - License: MIT

4. **`github.com/mattn/go-isatty`**
   - License: MIT

5. **`golang.org/x/sys`**
   - License: BSD-3-Clause
