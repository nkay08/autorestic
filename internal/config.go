package internal

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cupcakearmy/autorestic/internal/colors"
	"github.com/cupcakearmy/autorestic/internal/flags"
	"github.com/cupcakearmy/autorestic/internal/lock"
	"github.com/joho/godotenv"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const VERSION = "1.8.3"

type OptionMap map[string][]interface{}
type Options map[string]OptionMap

type Config struct {
	Version   string              `mapstructure:"version" yaml:"version"`
	Extras    interface{}         `mapstructure:"extras" yaml:"extras"`
	Locations map[string]Location `mapstructure:"locations" yaml:"locations"`
	Backends  map[string]Backend  `mapstructure:"backends" yaml:"backends"`
	Global    Options             `mapstructure:"global" yaml:"global"`
}

var once sync.Once
var config *Config

func exitConfig(err error, msg string) {
	if err != nil {
		colors.Error.Println(err)
	}
	if msg != "" {
		colors.Error.Println(msg)
	}
	lock.Unlock()
	os.Exit(1)
}

func GetConfig() *Config {

	if config == nil {
		once.Do(func() {
			if err := viper.ReadInConfig(); err == nil {
				absConfig, _ := filepath.Abs(viper.ConfigFileUsed())
				if !flags.CRON_LEAN {
					colors.Faint.Println("Using config: \t", absConfig)
				}
				// Load env file
				envFile := filepath.Join(filepath.Dir(absConfig), ".autorestic.env")
				err = godotenv.Load(envFile)
				if err == nil && !flags.CRON_LEAN {
					colors.Faint.Println("Using env:\t", envFile)
				}
			} else {
				text := err.Error()
				if strings.Contains(text, "no such file or directory") {
					cfgFileName := ".autorestic"
					colors.Error.Println(
						fmt.Sprintf(
							"cannot find configuration file '%s.yml' or '%s.yaml'.",
							cfgFileName, cfgFileName))
				} else {
					colors.Error.Println("could not load config file\n" + text)
				}
				os.Exit(1)
			}

			var versionConfig interface{}
			viper.UnmarshalKey("version", &versionConfig)
			if versionConfig == nil {
				exitConfig(nil, "no version specified in config file. please see docs on how to migrate")
			}
			version, ok := versionConfig.(int)
			if !ok {
				exitConfig(nil, "version specified in config file is not an int")
			} else {
				// Check for version
				if version != 2 {
					exitConfig(nil, "unsupported config version number. please check the docs for migration\nhttps://autorestic.vercel.app/migration/")
				}
			}

			config = &Config{}
			if err := viper.UnmarshalExact(config); err != nil {
				exitConfig(err, "Could not parse config file!")
			}
		})
	}
	return config
}

func GetPathRelativeToConfig(p string) (string, error) {
	if path.IsAbs(p) {
		return p, nil
	} else if strings.HasPrefix(p, "~") {
		home, err := homedir.Dir()
		return path.Join(home, strings.TrimPrefix(p, "~")), err
	} else {
		return path.Join(path.Dir(viper.ConfigFileUsed()), p), nil
	}
}

func (c *Config) Describe() {
	// Locations
	for name, l := range c.Locations {
		var tmp string
		colors.PrimaryPrint(`Location: "%s"`, name)

		tmp = ""
		for _, path := range l.From {
			tmp += fmt.Sprintf("\t%s %s\n", colors.Success.Sprint("←"), path)
		}
		colors.PrintDescription("From", tmp)

		tmp = ""
		for _, to := range l.To {
			tmp += fmt.Sprintf("\t%s %s\n", colors.Success.Sprint("→"), to)
		}
		colors.PrintDescription("To", tmp)

		tmp = ""
		for _, locDep := range l.DependsOn {
			tmp += fmt.Sprintf("\t%s %s\n", colors.Success.Sprint("→"), locDep)
		}
		colors.PrintDescription("Depends On", tmp)

		if l.Cron != "" {
			colors.PrintDescription("Cron", l.Cron)
		}

		tmp = ""
		hooks := map[string][]string{
			"PreValidate": l.Hooks.PreValidate,
			"Before":      l.Hooks.Before,
			"After":       l.Hooks.After,
			"Failure":     l.Hooks.Failure,
			"Success":     l.Hooks.Success,
		}
		for hook, commands := range hooks {
			if len(commands) > 0 {
				tmp += "\n\t" + hook
				for _, cmd := range commands {
					tmp += colors.Faint.Sprintf("\n\t  ▶ %s", cmd)
				}
			}
		}
		if tmp != "" {
			colors.PrintDescription("Hooks", tmp)
		}

		if len(l.Options) > 0 {
			tmp = ""
			for t, options := range l.Options {
				tmp += "\n\t" + t
				for option, values := range options {
					for _, value := range values {
						tmp += colors.Faint.Sprintf("\n\t  ✧ --%s=%s", option, value)
					}
				}
			}
			colors.PrintDescription("Options", tmp)
		}
	}

	// Backends
	for name, b := range c.Backends {
		colors.PrimaryPrint("Backend: \"%s\"", name)
		colors.PrintDescription("Type", b.Type)
		colors.PrintDescription("Path", b.Path)

		if len(b.Env) > 0 {
			tmp := ""
			for option, value := range b.Env {
				tmp += fmt.Sprintf("\n\t%s %s %s", colors.Success.Sprint("✧"), strings.ToUpper(option), colors.Faint.Sprint(value))
			}
			colors.PrintDescription("Env", tmp)
		}
	}
}

func CheckConfig() error {
	c := GetConfig()
	if c == nil {
		return fmt.Errorf("config could not be loaded/found")
	}
	if !CheckIfResticIsCallable() {
		return fmt.Errorf(`%s was not found. Install either with "autorestic install" or manually`, flags.RESTIC_BIN)
	}
	for name, backend := range c.Backends {
		backend.name = name
		if err := backend.validate(); err != nil {
			return err
		}
	}
	for name, location := range c.Locations {
		location.name = name
		if err := location.validate(); err != nil {
			return err
		}
	}

	if _, err := SortLocationsTopologicalFromMap(c.Locations); err != nil {
		return err
	}

	return nil
}

func GetLocationAdjacencyListFromLocationMap(locations map[string]Location, add_not_in_set bool) map[string][]string {
	locationsAsStrings := make(map[string][]string)
	for key, location := range locations {
		if _, ok := locationsAsStrings[key]; !ok {
			locationsAsStrings[key] = []string{}
		}
		for _, depLocStr := range location.DependsOn {
			// If the dependency is not in the set of locations, do not add it
			if _, ok := locations[depLocStr]; ok || add_not_in_set {
				locationsAsStrings[key] = append(locationsAsStrings[key], depLocStr)
			}
		}
	}
	return locationsAsStrings
}

func GetLocationAdjacencyListFromLocationStrings(locations []string, add_not_in_set bool) map[string][]string {
	locationsAsMap := make(map[string]Location)
	for _, locationString := range locations {
		if loc, ok := GetLocation(locationString); ok {
			locationsAsMap[locationString] = loc
		}
	}
	return GetLocationAdjacencyListFromLocationMap(locationsAsMap, add_not_in_set)
}

func SortLocationsTopologicalFromMap(locations map[string]Location) ([]string, error) {
	adjacencyList := GetLocationAdjacencyListFromLocationMap(locations, false)
	return TopologicalSort(adjacencyList, true)
}

func SortLocationsTopologicalFromStrings(locations []string) ([]string, error) {
	adjacencyList := GetLocationAdjacencyListFromLocationStrings(locations, false)
	return TopologicalSort(adjacencyList, true)
}

func GetAllOrSelected(cmd *cobra.Command, backends bool) ([]string, error) {
	var list []string
	if backends {
		for name := range config.Backends {
			list = append(list, name)
		}
	} else {
		for name := range config.Locations {
			list = append(list, name)
		}
	}

	all, _ := cmd.Flags().GetBool("all")
	if all {
		if backends {
			return list, nil
		} else {
			list, err := SortLocationsTopologicalFromStrings(list)
			return list, err
		}
	}

	var selected []string
	if backends {
		selected, _ = cmd.Flags().GetStringSlice("backend")
	} else {
		selected, _ = cmd.Flags().GetStringSlice("location")
	}
	for _, s := range selected {
		var splitted = strings.Split(s, "@")
		for _, l := range list {
			if l == splitted[0] {
				goto found
			}
		}
		if backends {
			return nil, fmt.Errorf("invalid backend \"%s\"", s)
		} else {
			return nil, fmt.Errorf("invalid location \"%s\"", s)
		}
	found:
	}

	if len(selected) == 0 {
		return selected, fmt.Errorf("nothing selected, aborting")
	}

	if backends {
		return selected, nil
	} else {
		selected, err := SortLocationsTopologicalFromStrings(selected)
		return selected, err
	}
}

// Return list of locations that are due to run based on the current time and each location's cron attribute.
// The list of locations will be topologically sorted.
func GetDueCronLocations(cmd *cobra.Command) ([]string, error) {
	var cronLocations []string
	var err error
	var allLocations []string

	if cmd != nil {
		allLocations, err = GetAllOrSelected(cmd, false)
	} else {
		allLocations, err = SortLocationsTopologicalFromMap(config.Locations)
	}

	if err != nil {
		return []string{}, err
	}

	for _, locationString := range allLocations {
		if location, ok := GetLocation(locationString); ok {
			runCron, err := location.CheckCron()
			if err != nil {
				return []string{}, err
			}
			if runCron {
				cronLocations = append(cronLocations, locationString)
			}
		}
	}

	return cronLocations, nil
}

func AddFlagsToCommand(cmd *cobra.Command, backend bool) {
	var usage string
	if backend {
		usage = "all backends"
	} else {
		usage = "all locations"
	}
	cmd.PersistentFlags().BoolP("all", "a", false, usage)
	if backend {
		cmd.PersistentFlags().StringSliceP("backend", "b", []string{}, "select backends")
	} else {
		cmd.PersistentFlags().StringSliceP("location", "l", []string{}, "select locations")
	}
}

func (c *Config) SaveConfig() error {
	file := viper.ConfigFileUsed()
	if err := CopyFile(file, file+".old"); err != nil {
		return err
	}
	colors.Secondary.Println("Saved a backup copy of your file next to the original.")

	viper.Set("backends", c.Backends)
	viper.Set("locations", c.Locations)

	return viper.WriteConfig()
}

func optionToString(option string) string {
	if !strings.HasPrefix(option, "-") {
		return "--" + option
	}
	return option
}

func appendOptionsToSlice(str *[]string, options OptionMap) {
	for key, values := range options {
		for _, value := range values {
			// Bool
			asBool, ok := value.(bool)
			if ok && asBool {
				*str = append(*str, optionToString(key))
				continue
			}
			*str = append(*str, optionToString(key), fmt.Sprint(value))
		}
	}
}

func getOptions(options Options, keys []string) []string {
	var selected []string
	for _, key := range keys {
		appendOptionsToSlice(&selected, options[key])
	}
	return selected
}

func combineBackendOptions(key string, b Backend) []string {
	// Priority: backend > global
	var options []string
	gFlags := getOptions(GetConfig().Global, []string{key})
	bFlags := getOptions(b.Options, []string{"all", key})
	options = append(options, gFlags...)
	options = append(options, bFlags...)
	return options
}

func combineAllOptions(key string, l Location, b Backend) []string {
	// Priority: location > backend > global
	var options []string
	gFlags := getOptions(GetConfig().Global, []string{key})
	bFlags := getOptions(b.Options, []string{"all", key})
	lFlags := getOptions(l.Options, []string{"all", key})
	options = append(options, gFlags...)
	options = append(options, bFlags...)
	options = append(options, lFlags...)
	return options
}
