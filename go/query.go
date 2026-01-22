package langlang

import (
	"fmt"
	"sync"
)

// QueryKey is the constraint for query keys - they must be comparable
// for use as map keys.
type QueryKey interface {
	comparable
}

// FilePath is a query key representing a file path for file-level queries.
type FilePath string

// DefKey is a query key for definition-level queries, identifying a
// specific definition within a grammar file.
type DefKey struct {
	File string
	Name string
}

// IdentifierLocation represents a reference to a rule in the grammar.
type IdentifierLocation struct {
	Name         string
	Location     SourceLocation
	IsDefinition bool // true if this is the definition site
}

// CallGraphData contains the call relationships between rules.
type CallGraphData struct {
	// Callers[A] = rules that call A (with locations)
	Callers map[string][]CallerInfo
	// Callees[A] = rules that A calls
	Callees map[string][]string
}

// CallerInfo describes a call site.
type CallerInfo struct {
	Name     string         // the calling rule's name
	Location SourceLocation // where the call occurs
}

// Query represents a computation that can be cached and tracked for
// dependencies. K is the key type (input) and V is the value type
// (output).
type Query[K QueryKey, V any] struct {
	Name    string
	Compute func(db *Database, key K) (V, error)
}

// queryID is a unique identifier for a cached query result, combining
// the query name with its key.
type queryID struct {
	queryName string
	key       any
}

// cachedValue holds a cached computation result along with metadata
// for invalidation.
type cachedValue struct {
	value    any
	err      error
	revision int
}

// Database is the central store for query results and dependency
// tracking. It manages caching, invalidation, and the query execution
// lifecycle.
type Database struct {
	mu sync.RWMutex

	// revision is incremented each time an input changes
	revision int

	// cache stores computed query results
	cache map[queryID]cachedValue

	// deps tracks which queries a given query depends on (forward deps)
	deps map[queryID][]queryID

	// rdeps tracks which queries depend on a given query (reverse deps)
	rdeps map[queryID][]queryID

	// activeQuery tracks the currently executing query for dependency recording
	activeQuery *queryID

	// config holds compiler configuration
	config *Config

	// loader is used for loading grammar files
	loader ImportLoader

	// fileIDs maps file paths to stable FileIDs for source location tracking
	fileIDs map[string]FileID

	// filePaths maps FileIDs back to paths (reverse of fileIDs)
	filePaths map[FileID]string

	// nextFileID is the next FileID to assign
	nextFileID FileID
}

// NewDatabase creates a new query database with the given
// configuration and import loader.
func NewDatabase(config *Config, loader ImportLoader) *Database {
	return &Database{
		revision:   0,
		cache:      make(map[queryID]cachedValue),
		deps:       make(map[queryID][]queryID),
		rdeps:      make(map[queryID][]queryID),
		config:     config,
		loader:     loader,
		fileIDs:    make(map[string]FileID),
		filePaths:  make(map[FileID]string),
		nextFileID: unknownFileID, // Start at -1, first file gets ID 0
	}
}

// InternFileID returns a stable FileID for the given path, creating one
// if it doesn't exist. This ensures the same path always gets the same ID.
func (db *Database) InternFileID(path string) FileID {
	db.mu.Lock()
	defer db.mu.Unlock()

	if id, ok := db.fileIDs[path]; ok {
		return id
	}

	db.nextFileID++
	id := db.nextFileID
	db.fileIDs[path] = id
	db.filePaths[id] = path
	return id
}

// FileIDToPath returns the file path for a given FileID, or empty string if not found.
func (db *Database) FileIDToPath(id FileID) string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.filePaths[id]
}

// AllFilePaths returns all registered file paths in FileID order.
func (db *Database) AllFilePaths() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.nextFileID < 0 {
		return nil
	}

	paths := make([]string, int(db.nextFileID)+1)
	for i := 0; i <= int(db.nextFileID); i++ {
		paths[i] = db.filePaths[FileID(i)]
	}
	return paths
}

// Config returns the database's configuration
func (db *Database) Config() *Config { return db.config }

// Loader returns the database's import loader
func (db *Database) Loader() ImportLoader { return db.loader }

// Revision returns the current database revision
func (db *Database) Revision() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.revision
}

// Get executes a query, returning a cached result if available and
// valid, or computing and caching a new result. It automatically
// tracks dependencies between queries.
func Get[K QueryKey, V any](db *Database, q *Query[K, V], key K) (V, error) {
	id := queryID{queryName: q.Name, key: key}

	db.mu.Lock()

	// Record dependency if we're inside another query
	if db.activeQuery != nil {
		parent := *db.activeQuery
		db.deps[parent] = append(db.deps[parent], id)
		db.rdeps[id] = append(db.rdeps[id], parent)
	}

	// Check cache
	if cached, ok := db.cache[id]; ok {
		db.mu.Unlock()
		if cached.err != nil {
			var zero V
			return zero, cached.err
		}
		return cached.value.(V), nil
	}

	// Set this as the active query for dependency tracking
	prevActive := db.activeQuery
	db.activeQuery = &id

	// Clear any stale dependencies from previous computations
	db.deps[id] = nil

	db.mu.Unlock()

	// Compute the value (outside the lock to allow nested queries)
	value, err := q.Compute(db, key)

	db.mu.Lock()
	// Restore previous active query
	db.activeQuery = prevActive

	// Cache the result
	db.cache[id] = cachedValue{
		value:    value,
		err:      err,
		revision: db.revision,
	}
	db.mu.Unlock()

	return value, err
}

// SetInput sets an input value directly in the cache and invalidates
// all dependent queries. This is used for "leaf" queries that
// represent external inputs (like file contents).
func SetInput[K QueryKey, V any](db *Database, q *Query[K, V], key K, value V) {
	id := queryID{queryName: q.Name, key: key}

	db.mu.Lock()
	defer db.mu.Unlock()

	db.revision++
	db.cache[id] = cachedValue{
		value:    value,
		err:      nil,
		revision: db.revision,
	}
	db.invalidateDependents(id)
}

// Invalidate removes a cached value and all its dependents from the
// cache.  This forces recomputation on the next query.
func Invalidate[K QueryKey, V any](db *Database, q *Query[K, V], key K) {
	id := queryID{queryName: q.Name, key: key}

	db.mu.Lock()
	defer db.mu.Unlock()

	db.invalidateWithDependents(id)
}

// invalidateDependents removes all queries that depend on the given
// query from the cache. Must be called with db.mu held
func (db *Database) invalidateDependents(id queryID) {
	dependents := db.rdeps[id]
	for _, dep := range dependents {
		delete(db.cache, dep)
		db.invalidateDependents(dep) // Recursively invalidate
	}
}

// invalidateWithDependents removes the given query and all its
// dependents from the cache. Must be called with db.mu held
func (db *Database) invalidateWithDependents(id queryID) {
	delete(db.cache, id)
	db.invalidateDependents(id)
}

// InvalidateAll clears all cached values, forcing full recomputation
func (db *Database) InvalidateAll() {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.revision++
	db.cache = make(map[queryID]cachedValue)
	db.deps = make(map[queryID][]queryID)
	db.rdeps = make(map[queryID][]queryID)
}

// InvalidateFile invalidates all queries related to a specific file.
// This is useful when a file's contents change.
func (db *Database) InvalidateFile(path string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.revision++

	// Invalidate ParsedGrammar for this file
	parsedID := queryID{queryName: "ParsedGrammar", key: FilePath(path)}
	db.invalidateWithDependents(parsedID)

	// Invalidate PosIndex for this file (reads raw content directly)
	posIndexID := queryID{queryName: "PosIndex", key: FilePath(path)}
	db.invalidateWithDependents(posIndexID)

	// Also invalidate any DefKey queries for this file
	for id := range db.cache {
		if defKey, ok := id.key.(DefKey); ok && defKey.File == path {
			db.invalidateWithDependents(id)
		}
	}
}

// Stats returns statistics about the query cache (mostly for debugging/testing).
func (db *Database) Stats() DatabaseStats {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return DatabaseStats{
		Revision:    db.revision,
		CachedCount: len(db.cache),
		DepsCount:   len(db.deps),
	}
}

// DatabaseStats holds statistics about the query database.
type DatabaseStats struct {
	Revision    int
	CachedCount int
	DepsCount   int
}

func (s DatabaseStats) String() string {
	return fmt.Sprintf("Database{revision=%d, cached=%d, deps=%d}",
		s.Revision, s.CachedCount, s.DepsCount)
}
