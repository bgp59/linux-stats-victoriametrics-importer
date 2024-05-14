package lsvmi

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	LOGGER_CONFIG_USE_JSON_DEFAULT            = true
	LOGGER_CONFIG_LEVEL_DEFAULT               = "info"
	LOGGER_CONFIG_DISBALE_REPORT_FILE_DEFAULT = false

	LOGGER_DEFAULT_LEVEL    = logrus.InfoLevel
	LOGGER_TIMESTAMP_FORMAT = time.RFC3339Nano
	// Extra field added for component sub loggers:
	LOGGER_COMPONENT_FIELD_NAME = "comp"
)

// CollectableLog interface for logurs.Log (see
// ../internal/testutil/log_collector.go):
type CollectableLogursLog struct {
	logrus.Logger
	// Cache the condition of being enabled for debug or not. Various sections
	// of  the code may test this condition before doing more expensive actions,
	// such as formatting debug info, so it pays off to make it as efficient as
	// possible:
	IsEnabledForDebug bool
}

func (log *CollectableLogursLog) GetOutput() io.Writer {
	return log.Out
}

func (log *CollectableLogursLog) GetLevel() any {
	return log.Logger.GetLevel()
}

func (log *CollectableLogursLog) SetLevel(level any) {
	if level, ok := level.(logrus.Level); ok {
		log.Logger.SetLevel(level)
		log.IsEnabledForDebug = log.IsLevelEnabled(logrus.DebugLevel)
	}
}

type LoggerConfig struct {
	UseJson           bool   `yaml:"use_json"`
	Level             string `yaml:"level"`
	DisableReportFile bool   `yaml:"disable_report_file"`
}

func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		UseJson:           LOGGER_CONFIG_USE_JSON_DEFAULT,
		Level:             LOGGER_CONFIG_LEVEL_DEFAULT,
		DisableReportFile: LOGGER_CONFIG_DISBALE_REPORT_FILE_DEFAULT,
	}
}

var loggerUseJsonArg = NewBoolFlagCheckUsed(
	"log-json-format",
	"Enable log in JSON format",
)

var loggerLevelArg = NewStringFlagCheckUsed(
	"log-level",
	LOGGER_DEFAULT_LEVEL.String(),
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
		funcFile = &LogFuncFilePair{
			"", //f.Function,
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
	DisableQuote:     false,
	FullTimestamp:    true,
	TimestampFormat:  LOGGER_TIMESTAMP_FORMAT,
	CallerPrettyfier: logFunctionFileCache.LogCallerPrettyfier,
	DisableSorting:   false,
	SortingFunc:      LogSortFieldKeys,
}

var LogJsonFormatter = &logrus.JSONFormatter{
	TimestampFormat:  LOGGER_TIMESTAMP_FORMAT,
	CallerPrettyfier: logFunctionFileCache.LogCallerPrettyfier,
}

var Log = &CollectableLogursLog{
	Logger: logrus.Logger{
		Out: os.Stderr,
		//Hooks:        make(logrus.LevelHooks),
		Formatter:    LogTextFormatter,
		Level:        LOGGER_DEFAULT_LEVEL,
		ReportCaller: true,
	},
}

func GetLogLevelNames() []string {
	levelNames := make([]string, len(logrus.AllLevels))
	for i, level := range logrus.AllLevels {
		levelNames[i] = level.String()
	}
	return levelNames
}

func init() {
	root, err := GetSourceRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		if root != "/" {
			logSourceRoot = root + "/"
		} else {
			logSourceRoot = root
		}
	}
}

// Set the logger based on config overridden by command line args, if the latter
// were used:
func SetLogger(cfg any) error {
	var (
		levelName string
		logCfg    *LoggerConfig
	)

	switch cfg := cfg.(type) {
	case *LoggerConfig:
		logCfg = cfg
	case *LsvmiConfig:
		logCfg = cfg.LoggerConfig
	case nil:
		logCfg = DefaultLoggerConfig()
	default:
		return fmt.Errorf("cfg: %T invalid type", cfg)
	}

	levelName = logCfg.Level
	if levelName != "" {
		level, err := logrus.ParseLevel(levelName)
		if err != nil {
			return err
		}
		Log.SetLevel(level)
	}

	if loggerUseJsonArg.Value {
		Log.SetFormatter(LogJsonFormatter)
	} else {
		Log.SetFormatter(LogTextFormatter)
	}

	Log.SetReportCaller(!logCfg.DisableReportFile)

	return nil
}

func NewCompLogger(compName string) *logrus.Entry {
	return Log.WithField(LOGGER_COMPONENT_FIELD_NAME, compName)
}
