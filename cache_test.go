package tmdbankigenerator

/*
// createTestFile creates a file with initial JSON-encoded content in a temp directory.
func createTestFile(t *testing.T, content any) string {
	t.Helper()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test_cache.json")

	bytes, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("failed to marshal initial content: %v", err)
	}

	if err := os.WriteFile(filePath, bytes, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	return filePath
}

// parseLastJSONBlob reads the entire file, and returns the last valid JSON object
// (because the file might contain multiple appended JSON blobs).
func parseLastJSONBlob(t *testing.T, filePath string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// The file may contain multiple JSON objects appended one after another.
	// We'll try to parse from the end. We'll do something simplistic:
	blobs := make([]map[string]any, 0)
	decoder := json.NewDecoder(stringToReader(string(data)))

	for {
		var m map[string]any
		if err := decoder.Decode(&m); err != nil {
			break // we assume we hit EOF or a decode error
		}
		blobs = append(blobs, m)
	}

	if len(blobs) == 0 {
		return nil
	}
	return blobs[len(blobs)-1] // last blob
}

// stringToReader is a convenience to make a string into an io.Reader
func stringToReader(s string) *stringReader {
	return &stringReader{str: s}
}

type stringReader struct {
	str string
	pos int
}

func (sr *stringReader) Read(p []byte) (n int, err error) {
	if sr.pos >= len(sr.str) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, sr.str[sr.pos:])
	sr.pos += n
	return n, nil
}

// TestNewCache tests that we can properly create a new cache from a file.
func TestNewCache(t *testing.T) {
	initial := map[string]string{
		"hello": "world",
		"foo":   "bar",
	}

	filePath := createTestFile(t, initial)

	cache, err := NewCache[string, string](filePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Confirm existing keys
	ok, val := cache.Get("hello")
	if !ok {
		t.Fatalf("expected key 'hello' to exist")
	}
	if val != "world" {
		t.Errorf("expected 'world', got %s", val)
	}

	ok, val = cache.Get("foo")
	if !ok {
		t.Fatalf("expected key 'foo' to exist")
	}
	if val != "bar" {
		t.Errorf("expected 'bar', got %s", val)
	}

	// Confirm a non-existent key
	ok, _ = cache.Get("no-such-key")
	if ok {
		t.Errorf("did not expect key 'no-such-key' to exist")
	}
}

// TestCacheSetAndGet tests that Set overwrites/creates a key and Get retrieves it.
func TestCacheSetAndGet(t *testing.T) {
	filePath := createTestFile(t, map[string]string{
		"initialKey": "initialValue",
	})

	cache, err := NewCache[string, string](filePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Set a new key
	cache.Set("hello", "world")

	// Let the background file write complete (a small sleep is usually not ideal in production tests,
	// but okay in a simple example).
	time.Sleep(100 * time.Millisecond)

	// Check in-memory value
	ok, val := cache.Get("hello")
	if !ok {
		t.Fatalf("expected key 'hello' to exist")
	}
	if val != "world" {
		t.Errorf("expected 'world', got %s", val)
	}

	// Also check last blob in the file to ensure it’s persisted
	lastBlob := parseLastJSONBlob(t, filePath)
	if lastBlob == nil {
		t.Fatalf("could not find any JSON objects in file")
	}
	if lastBlob["hello"] != "world" {
		t.Errorf("expected file's last JSON blob to have 'hello' = 'world'; got %v", lastBlob["hello"])
	}
}

// TestCacheConcurrency tests concurrent writes to ensure there are no data races
// and that the final in-memory data is correct.
func TestCacheConcurrency(t *testing.T) {
	filePath := createTestFile(t, map[string]string{
		"concurrent": "test",
	})

	cache, err := NewCache[string, string](filePath)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key_%d", idx)
			val := fmt.Sprintf("val_%d", idx)
			cache.Set(key, val)
		}(i)
	}
	wg.Wait()

	// Wait a bit to let background file writes complete
	time.Sleep(300 * time.Millisecond)

	// Check in-memory results
	for i := 0; i < numGoroutines; i++ {
		key := fmt.Sprintf("key_%d", i)
		val := fmt.Sprintf("val_%d", i)
		ok, gotVal := cache.Get(key)
		if !ok {
			t.Errorf("key %s not found in cache", key)
			continue
		}
		if gotVal != val {
			t.Errorf("expected %s for key %s, got %s", val, key, gotVal)
		}
	}

	// Confirm that the last blob in the file also has them. (Depending on how many sets
	// were appended, we'll just check the last one.)
	lastBlob := parseLastJSONBlob(t, filePath)
	if lastBlob == nil {
		t.Fatalf("could not find any JSON objects in file after concurrency test")
	}

	for i := 0; i < numGoroutines; i++ {
		key := fmt.Sprintf("key_%d", i)
		val := fmt.Sprintf("val_%d", i)

		if got, ok := lastBlob[key]; !ok {
			t.Errorf("file last JSON blob missing key %s", key)
		} else if got != val {
			t.Errorf("expected file last JSON blob's %s = %s, got %v", key, val, got)
		}
	}
}
*/
