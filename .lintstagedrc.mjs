// Root lint-staged config. Files under frontend/ use frontend/.lintstagedrc.mjs instead.
export default {
  // Format staged Go files, then vet and test the backend once (rules can break from unstaged code too).
  "backend/**/*.go": ["gofmt -w", () => "sh -c 'cd backend && go vet ./... && go test ./...'"],
  "backend/**/*.proto": () => "sh -c 'cd backend && PATH=$HOME/go/bin:$PATH buf lint'",
  "*.sh": (files) => files.map((f) => `sh -n ${f}`),
}
