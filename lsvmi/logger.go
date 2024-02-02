package lsvmi

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	DEFAULT_LOG_LEVEL = logrus.InfoLevel
	// Extra field added for component sub loggers:
	LOGGER_COMPONENT_FIELD_NAME = "comp"
)

type LoggerConfig struct {
	UseJson bool
	Level   string
}

var loggerUseJsonArg = NewBoolFlagCheckUsed(
	"log-json-format",
	"Enable log in JSON format",
)

var loggerLevelArg = NewStringFlagCheckUsed(
	"log-level",
	DEFAULT_LOG_LEVEL.String(),
	fmt.Sprintf(`
	Set log level, it should be one of the %s values. 
	`, GetLogLevelNames()),
)

var logSourceRoot string

func GetSourceRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot determine source root: runtime.Caller(0) failed")
	}
	return path.Dir(path.Dir(file)), nil
}

func init() {
	root, err := GetSourceRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {

	}
	if root != "/" {
		logSourceRoot = root + "/"
	} else {
		logSourceRoot = root
	}
}

// Maintain a cache for caller PC -> (file:line#, function) to speed up the
// formatting:
type LogFuncFilePair struct {
	function string
	file     string
}

type LogFuncFileCache struct {
	m             *sync.Mutex
	funcFileCache map[uintptr]*LogFuncFilePair
}

// Return the function name and filename:line# info from the frame. The filename is
// relative to the source root dir.
func (c *LogFuncFileCache) LogCallerPrettyfier(f *runtime.Frame) (function string, file string) {
	c.m.Lock()
	defer c.m.Unlock()
	funcFile := c.funcFileCache[f.PC]
	if funcFile == nil {
		filename := ""
		if logSourceRoot != "" && strings.HasPrefix(f.File, logSourceRoot) {
			filename = f.File[len(logSourceRoot):]
		} else {
			_, filename = path.Split(f.File)
		}
		function := ""
		funcFile = &LogFuncFilePair{
			function,
			fmt.Sprintf("%s:%d", filename, f.Line),
		}
		c.funcFileCache[f.PC] = funcFile
	}
	return funcFile.function, funcFile.file
}

var logFunctionFileCache = &LogFuncFileCache{
	m:             &sync.Mutex{},
	funcFileCache: make(map[uintptr]*LogFuncFilePair),
}

var LogFieldKeySortOrder = map[string]int{
	// The desired order is time, level, file, func, other fields sorted
	// alphabetically and msg. Use negative numbers for the fields preceding
	// `other' to capitalize on the fact that any of the latter will return 0 at
	// lookup.
	logrus.FieldKeyTime:         -5,
	logrus.FieldKeyLevel:        -4,
	LOGGER_COMPONENT_FIELD_NAME: -3,
	logrus.FieldKeyFile:         -2,
	logrus.FieldKeyFunc:         -1,
	logrus.FieldKeyMsg:          1,
}

type LogFieldKeySortable struct {
	keys []string
}

func (d *LogFieldKeySortable) Len() int {
	return len(d.keys)
}

func (d *LogFieldKeySortable) Less(i, j int) bool {
	key_i, key_j := d.keys[i], d.keys[j]
	order_i, order_j := LogFieldKeySortOrder[key_i], LogFieldKeySortOrder[key_j]
	if order_i != 0 || order_j != 0 {
		return order_i < order_j
	}
	return strings.Compare(key_i, key_j) == -1
}

func (d *LogFieldKeySortable) Swap(i, j int) {
	d.keys[i], d.keys[j] = d.keys[j], d.keys[i]
}

func LogSortFieldKeys(keys []string) {
	sort.Sort(&LogFieldKeySortable{keys})
}

var LogTextFormatter = &logrus.TextFormatter{
	DisableColors:    true,
	FullTimestamp:    true,
	CallerPrettyfier: logFunctionFileCache.LogCallerPrettyfier,
	DisableSorting:   false,
	SortingFunc:      LogSortFieldKeys,
}

var LogJsonFormatter = &logrus.JSONFormatter{
	CallerPrettyfier: logFunctionFileCache.LogCallerPrettyfier,
}

var Log = &logrus.Logger{
	ReportCaller: true,
	Out:          os.Stderr,
	Formatter:    LogTextFormatter,
	Hooks:        make(logrus.LevelHooks),
	Level:        DEFAULT_LOG_LEVEL,
}

func GetLogLevelNames() []string {
	levelNames := make([]string, len(logrus.AllLevels))
	for i, level := range logrus.AllLevels {
		levelNames[i] = level.String()
	}
	return levelNames
}

// Set the logger based on config overridden by command line args, if the latter
// were used:
func SetLogger(cfg *LoggerConfig) error {
	var levelName string
	if loggerLevelArg.Used {
		levelName = loggerLevelArg.Value
	} else if cfg != nil {
		levelName = cfg.Level
	}
	if levelName != "" {
		level, err := logrus.ParseLevel(levelName)
		if err != nil {
			return err
		}
		Log.SetLevel(level)
	}
	if loggerUseJsonArg.Used || (cfg != nil && cfg.UseJson) {
		Log.SetFormatter(LogJsonFormatter)
	}
	return nil
}
