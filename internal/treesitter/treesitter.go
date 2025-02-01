package treesitter

// treesitter.go serves as the package entry point and re-exports commonly used functionality.

// For example, you can re-export the NewDummyParser function:
var NewParser = NewDummyParser

// You can also re-export update triggers if desired.
// (Clients of this package can call treesitter.OnCommit, etc.)
