// Configuration for the stats collector

package lsvmi

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/go-yaml/yaml"
)

// The configuration is stored in one object to make it easy to load it from a
// file. Most of the configuration parameters are based on the file settings and
// a few can be overridden by command line arguments.
//
// The decreasing order of precedence for parameter values:
//   - command line arg (if applicable)
//   - config file
//   - built-in default
//
// Notes:
//  1. Each component will have its specific configuration, which may be
//     defined elsewhere, for instance in the files(s) providing the implementation.
// 2. The object will be loaded from a YAML file, therefore all configuration
//    parameters should be public and they should have tag annotations.

const (
	GLOBAL_CONFIG_INSTANCE_DEFAULT           = "lsvmi"
	GLOBAL_CONFIG_USE_SHORT_HOSTNAME_DEFAULT = true
	GLOBAL_CONFIG_PROCFS_ROOT_DEFAULT        = "/proc"
)

type LsvmiConfig struct {
	GlobalConfig           *GlobalConfig           `yaml:"global_config"`
	ProcStatMetricsConfig  *ProcStatMetricsConfig  `yaml:"proc_stat_metrics_config"`
	InternalMetricsConfig  *InternalMetricsConfig  `yaml:"internal_metrics_config"`
	SchedulerConfig        *SchedulerConfig        `yaml:"scheduler_config"`
	CompressorPoolConfig   *CompressorPoolConfig   `yaml:"compressor_pool_config"`
	HttpEndpointPoolConfig *HttpEndpointPoolConfig `yaml:"http_endpoint_pool_config"`
	LoggerConfig           *LoggerConfig           `yaml:"log_config"`
}

type GlobalConfig struct {
	// All metrics have the instance and hostname labels.

	// The instance name, default "lsvmi". It may be overridden by --instance
	// command line arg.
	Instance string `yaml:"instance"`

	// Whether to use short hostname or not as the value for hostname label.
	// Typically the hostname is determined from the hostname system call and if
	// the flag below is in effect, it is stripped of domain part. However if
	// the hostname is overridden by --hostname command line arg, that value is
	// used as-is.
	UseShortHostname bool `yaml:"use_short_hostname"`

	// procfs root. It may be overridden by --procfs-root command line arg.
	ProcfsRoot string `yaml:"procfs_root"`
}

var ErrConfigFileArgNotProvided = errors.New("config file arg not provided")

var lsvmiConfigFile = flag.String(
	"config",
	"",
	`Config file to load`,
)

var hostnameArg = flag.String(
	"hostname",
	"",
	FormatFlagUsage(`
	Set the hostname to use as value for the metric label hostname. This
	overrides the value returned by hostname syscall.
	`),
)

var instanceArg = flag.String(
	"instance",
	"",
	"Override the config `instance` setting",
)

var procfsRootArg = flag.String(
	"procfs-root",
	"",
	"Override the config `procfs_root` setting",
)

func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		Instance:         GLOBAL_CONFIG_INSTANCE_DEFAULT,
		UseShortHostname: GLOBAL_CONFIG_USE_SHORT_HOSTNAME_DEFAULT,
		ProcfsRoot:       GLOBAL_CONFIG_PROCFS_ROOT_DEFAULT,
	}
}

func DefaultLsvmiConfig() *LsvmiConfig {
	return &LsvmiConfig{
		GlobalConfig:           DefaultGlobalConfig(),
		ProcStatMetricsConfig:  DefaultProcStatMetricsConfig(),
		InternalMetricsConfig:  DefaultInternalMetricsConfig(),
		SchedulerConfig:        DefaultSchedulerConfig(),
		CompressorPoolConfig:   DefaultCompressorPoolConfig(),
		HttpEndpointPoolConfig: DefaultHttpEndpointPoolConfig(),
	}
}

func LoadLsvmiConfig(cfgFile string) (*LsvmiConfig, error) {
	f, err := os.Open(cfgFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	cfg := DefaultLsvmiConfig()
	err = decoder.Decode(cfg)
	if err != nil {
		return nil, fmt.Errorf("file: %q: %v", cfgFile, err)
	}
	return cfg, nil
}

func LoadLsvmiConfigFromArgs() (*LsvmiConfig, error) {
	if *lsvmiConfigFile == "" {
		return nil, ErrConfigFileArgNotProvided
	}
	cfg, err := LoadLsvmiConfig(*lsvmiConfigFile)
	if err != nil {
		return nil, err
	}

	// Apply command line overrides:
	if *instanceArg != "" {
		cfg.GlobalConfig.Instance = *instanceArg
	}
	if *procfsRootArg != "" {
		cfg.GlobalConfig.ProcfsRoot = *procfsRootArg
	}
	if loggerUseJsonArg.Used {
		cfg.LoggerConfig.UseJson = loggerUseJsonArg.Value
	}
	if loggerLevelArg.Used {
		cfg.LoggerConfig.Level = loggerLevelArg.Value
	}

	return cfg, nil
}
