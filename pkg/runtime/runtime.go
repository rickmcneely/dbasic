package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func init() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())
}

// DBasic runtime support functions

// --- Input Functions ---

// Input reads a line from stdin with optional prompt
func Input(prompt string) string {
	if prompt != "" {
		fmt.Print(prompt)
	}
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}

// InputInt reads an integer from stdin
func InputInt(prompt string) int64 {
	s := Input(prompt)
	val, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return val
}

// InputFloat reads a float from stdin
func InputFloat(prompt string) float64 {
	s := Input(prompt)
	val, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return val
}

// --- String Functions ---

// Len returns the length of a string
func Len(s string) int32 {
	return int32(len(s))
}

// Left returns the leftmost n characters
func Left(s string, n int32) string {
	if n <= 0 {
		return ""
	}
	if int(n) >= len(s) {
		return s
	}
	return s[:n]
}

// Right returns the rightmost n characters
func Right(s string, n int32) string {
	if n <= 0 {
		return ""
	}
	if int(n) >= len(s) {
		return s
	}
	return s[len(s)-int(n):]
}

// Mid returns a substring starting at position start with length ln
func Mid(s string, start, ln int32) string {
	if start < 1 {
		start = 1
	}
	startIdx := int(start) - 1
	if startIdx >= len(s) {
		return ""
	}
	endIdx := startIdx + int(ln)
	if endIdx > len(s) {
		endIdx = len(s)
	}
	return s[startIdx:endIdx]
}

// Instr finds the position of substring in string (1-based)
func Instr(s, substr string) int32 {
	idx := strings.Index(s, substr)
	if idx == -1 {
		return 0
	}
	return int32(idx + 1)
}

// InstrRev finds the last position of substring in string (1-based)
func InstrRev(s, substr string) int32 {
	idx := strings.LastIndex(s, substr)
	if idx == -1 {
		return 0
	}
	return int32(idx + 1)
}

// UCase converts to uppercase
func UCase(s string) string {
	return strings.ToUpper(s)
}

// LCase converts to lowercase
func LCase(s string) string {
	return strings.ToLower(s)
}

// Trim removes leading and trailing whitespace
func Trim(s string) string {
	return strings.TrimSpace(s)
}

// LTrim removes leading whitespace
func LTrim(s string) string {
	return strings.TrimLeft(s, " \t\r\n")
}

// RTrim removes trailing whitespace
func RTrim(s string) string {
	return strings.TrimRight(s, " \t\r\n")
}

// Replace replaces all occurrences of old with new
func Replace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

// Split splits a string by delimiter and returns an array
func Split(s, delim string) []string {
	return strings.Split(s, delim)
}

// Join joins an array of strings with a delimiter
func Join(arr []string, delim string) string {
	return strings.Join(arr, delim)
}

// Space returns a string of n spaces
func Space(n int32) string {
	return strings.Repeat(" ", int(n))
}

// String returns a string of n copies of character
func String_(n int32, char string) string {
	if len(char) == 0 {
		return ""
	}
	return strings.Repeat(string(char[0]), int(n))
}

// Reverse reverses a string
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// --- Conversion Functions ---

// Str converts a number to string
func Str(val interface{}) string {
	return fmt.Sprintf("%v", val)
}

// Val converts a string to float64
func Val(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

// Hex converts a number to hexadecimal string
func Hex(val int64) string {
	return fmt.Sprintf("%X", val)
}

// Oct converts a number to octal string
func Oct(val int64) string {
	return fmt.Sprintf("%o", val)
}

// Bin converts a number to binary string
func Bin(val int64) string {
	return fmt.Sprintf("%b", val)
}

// Int converts to int32
func Int(val interface{}) int32 {
	switch v := val.(type) {
	case int:
		return int32(v)
	case int32:
		return v
	case int64:
		return int32(v)
	case float32:
		return int32(v)
	case float64:
		return int32(v)
	case string:
		i, _ := strconv.ParseInt(v, 10, 32)
		return int32(i)
	default:
		return 0
	}
}

// Lng converts to int64
func Lng(val interface{}) int64 {
	switch v := val.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		i, _ := strconv.ParseInt(v, 10, 64)
		return i
	default:
		return 0
	}
}

// Sng converts to float32
func Sng(val interface{}) float32 {
	switch v := val.(type) {
	case int:
		return float32(v)
	case int32:
		return float32(v)
	case int64:
		return float32(v)
	case float32:
		return v
	case float64:
		return float32(v)
	case string:
		f, _ := strconv.ParseFloat(v, 32)
		return float32(f)
	default:
		return 0
	}
}

// Dbl converts to float64
func Dbl(val interface{}) float64 {
	switch v := val.(type) {
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

// Bool converts to boolean
func Bool(val interface{}) bool {
	switch v := val.(type) {
	case bool:
		return v
	case int, int32, int64, float32, float64:
		return v != 0
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		return lower == "true" || lower == "yes" || lower == "1"
	default:
		return false
	}
}

// --- Byte/String Functions ---

// Encode converts a STRING to BYTES (UTF-8 encoding)
func Encode(s string) []byte {
	return []byte(s)
}

// Decode converts BYTES to STRING (UTF-8 decoding)
func Decode(b []byte) string {
	return string(b)
}

// MakeBytes creates a byte array of the specified size
func MakeBytes(size int32) []byte {
	return make([]byte, size)
}

// LenBytes returns the length of a byte array
func LenBytes(b []byte) int32 {
	return int32(len(b))
}

// --- Math Functions ---

// Abs returns the absolute value
func Abs(val float64) float64 {
	return math.Abs(val)
}

// Sqr returns the square root
func Sqr(val float64) float64 {
	return math.Sqrt(val)
}

// Sin returns the sine
func Sin(val float64) float64 {
	return math.Sin(val)
}

// Cos returns the cosine
func Cos(val float64) float64 {
	return math.Cos(val)
}

// Tan returns the tangent
func Tan(val float64) float64 {
	return math.Tan(val)
}

// Atn returns the arctangent
func Atn(val float64) float64 {
	return math.Atan(val)
}

// Atn2 returns the arctangent of y/x
func Atn2(y, x float64) float64 {
	return math.Atan2(y, x)
}

// Log returns the natural logarithm
func Log(val float64) float64 {
	return math.Log(val)
}

// Log10 returns the base-10 logarithm
func Log10(val float64) float64 {
	return math.Log10(val)
}

// Exp returns e^x
func Exp(val float64) float64 {
	return math.Exp(val)
}

// Pow returns x^y
func Pow(x, y float64) float64 {
	return math.Pow(x, y)
}

// Sgn returns the sign of a number (-1, 0, or 1)
func Sgn(val float64) int32 {
	if val < 0 {
		return -1
	}
	if val > 0 {
		return 1
	}
	return 0
}

// Fix truncates toward zero
func Fix(val float64) int64 {
	return int64(val)
}

// Floor returns the largest integer <= val
func Floor(val float64) int64 {
	return int64(math.Floor(val))
}

// Ceil returns the smallest integer >= val
func Ceil(val float64) int64 {
	return int64(math.Ceil(val))
}

// Round rounds to nearest integer
func Round(val float64) int64 {
	return int64(math.Round(val))
}

// Min returns the minimum of two values
func Min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two values
func Max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// Clamp returns val clamped between min and max
func Clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// PI returns the value of pi
func PI() float64 {
	return math.Pi
}

// --- Random Number Functions ---

// Rnd returns a random float64 between 0 and 1
func Rnd() float64 {
	return rand.Float64()
}

// RndInt returns a random integer between 0 and max-1
func RndInt(max int32) int32 {
	return int32(rand.Intn(int(max)))
}

// RndRange returns a random integer between min and max (inclusive)
func RndRange(min, max int32) int32 {
	return min + int32(rand.Intn(int(max-min+1)))
}

// Randomize seeds the random number generator
func Randomize(seed int64) {
	rand.Seed(seed)
}

// --- Date/Time Functions ---

// Timer returns the number of seconds since midnight
func Timer() float64 {
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return now.Sub(midnight).Seconds()
}

// Now returns the current Unix timestamp
func Now() int64 {
	return time.Now().Unix()
}

// Date returns the current date as a string (YYYY-MM-DD)
func Date() string {
	return time.Now().Format("2006-01-02")
}

// Time returns the current time as a string (HH:MM:SS)
func Time_() string {
	return time.Now().Format("15:04:05")
}

// Year returns the current year
func Year() int32 {
	return int32(time.Now().Year())
}

// Month returns the current month (1-12)
func Month() int32 {
	return int32(time.Now().Month())
}

// Day returns the current day of month
func Day() int32 {
	return int32(time.Now().Day())
}

// Hour returns the current hour (0-23)
func Hour() int32 {
	return int32(time.Now().Hour())
}

// Minute returns the current minute (0-59)
func Minute() int32 {
	return int32(time.Now().Minute())
}

// Second returns the current second (0-59)
func Second() int32 {
	return int32(time.Now().Second())
}

// Weekday returns the day of week (0=Sunday, 6=Saturday)
func Weekday() int32 {
	return int32(time.Now().Weekday())
}

// Sleep pauses execution for specified milliseconds
func Sleep(ms int32) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// --- JSON Functions ---

// JSONParse parses a JSON string into a map
func JSONParse(s string) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal([]byte(s), &result)
	return result
}

// JSONStringify converts a map to JSON string
func JSONStringify(data map[string]interface{}) string {
	b, _ := json.Marshal(data)
	return string(b)
}

// JSONPretty converts a map to pretty-printed JSON string
func JSONPretty(data map[string]interface{}) string {
	b, _ := json.MarshalIndent(data, "", "  ")
	return string(b)
}

// JSONGet retrieves a value from a JSON map by path
func JSONGet(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		default:
			return nil
		}
	}

	return current
}

// JSONSet sets a value in a JSON map by path
func JSONSet(data map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := data

	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if _, ok := current[part]; !ok {
			current[part] = make(map[string]interface{})
		}
		current = current[part].(map[string]interface{})
	}

	current[parts[len(parts)-1]] = value
}

// --- Array Functions ---

// ArrayLen returns the length of an array
func ArrayLen(arr interface{}) int32 {
	switch v := arr.(type) {
	case []interface{}:
		return int32(len(v))
	case []int32:
		return int32(len(v))
	case []int64:
		return int32(len(v))
	case []float64:
		return int32(len(v))
	case []string:
		return int32(len(v))
	case []bool:
		return int32(len(v))
	default:
		return 0
	}
}

// --- ASCII Functions ---

// Asc returns the ASCII code of the first character
func Asc(s string) int32 {
	if len(s) == 0 {
		return 0
	}
	return int32(s[0])
}

// Chr returns the character for an ASCII code
func Chr(code int32) string {
	return string(rune(code))
}

// --- File Functions ---

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ReadFile reads entire file contents
func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile writes string to file
func WriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// AppendFile appends string to file
func AppendFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

// DeleteFile deletes a file
func DeleteFile(path string) error {
	return os.Remove(path)
}

// CopyFile copies a file
func CopyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}

// MkDir creates a directory
func MkDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// RmDir removes a directory
func RmDir(path string) error {
	return os.RemoveAll(path)
}

// ListDir lists files in a directory
func ListDir(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}

// GetCwd returns the current working directory
func GetCwd() string {
	dir, _ := os.Getwd()
	return dir
}

// SetCwd changes the current working directory
func SetCwd(path string) error {
	return os.Chdir(path)
}

// BaseName returns the base name of a path
func BaseName(path string) string {
	return filepath.Base(path)
}

// DirName returns the directory name of a path
func DirName(path string) string {
	return filepath.Dir(path)
}

// JoinPath joins path elements
func JoinPath(parts ...string) string {
	return filepath.Join(parts...)
}

// --- Environment Functions ---

// Environ gets an environment variable
func Environ(name string) string {
	return os.Getenv(name)
}

// SetEnviron sets an environment variable
func SetEnviron(name, value string) error {
	return os.Setenv(name, value)
}

// GetArgs returns command line arguments
func GetArgs() []string {
	return os.Args
}

// Exit terminates the program with an exit code
func Exit(code int32) {
	os.Exit(int(code))
}

// --- Error Handling ---

// Error represents a runtime error
type Error struct {
	Message string
	Code    int
}

func (e *Error) Error() string {
	return e.Message
}

// NewError creates a new error
func NewError(message string) *Error {
	return &Error{Message: message, Code: 1}
}

// IsError checks if a value is an error
func IsError(val interface{}) bool {
	_, ok := val.(error)
	return ok
}
